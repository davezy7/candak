package downloader

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"candak/internal/events"
	"candak/internal/models"
	"candak/internal/planner"
	"candak/internal/prober"
	"candak/internal/storage/sqlite"
)

func TestDownloaderManagerIntegration(t *testing.T) {
	ctx := context.Background()
	payload := bytes.Repeat([]byte("SOLID_PRINCIPLES_DOWNLOAD_012345"), 4096) // 128 KB

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Disposition", `attachment; filename="solid_data.bin"`)

		rangeHeader := r.Header.Get("Range")
		if rangeHeader == "" {
			w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
			if r.Method == "HEAD" {
				w.WriteHeader(http.StatusOK)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(payload)
			return
		}

		if !strings.HasPrefix(rangeHeader, "bytes=") {
			w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
			return
		}

		parts := strings.Split(strings.TrimPrefix(rangeHeader, "bytes="), "-")
		start, _ := strconv.ParseInt(parts[0], 10, 64)
		end := int64(len(payload) - 1)
		if len(parts) > 1 && parts[1] != "" {
			end, _ = strconv.ParseInt(parts[1], 10, 64)
		}

		data := payload[start : end+1]
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(payload)))
		w.Header().Set("Content-Length", strconv.Itoa(len(data)))
		w.WriteHeader(http.StatusPartialContent)
		if r.Method != "HEAD" {
			_, _ = w.Write(data)
		}
	}))
	defer server.Close()

	tempDir, err := os.MkdirTemp("", "candak-mgr-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	repo, err := sqlite.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to init sqlite repo: %v", err)
	}
	defer repo.Close()

	// Dependency Injection verification
	mgr := NewManager(Config{
		Repository:   repo,
		Prober:       prober.NewHTTPProber(prober.WithHTTPClient(server.Client())),
		ChunkPlanner: planner.NewDefaultChunkPlanner(),
		Broadcaster:  events.NewNullBroadcaster(),
		HTTPClient:   server.Client(),
	})

	// 1. Probe URL
	probeRes, err := mgr.ProbeURL(ctx, server.URL+"/myfile.bin")
	if err != nil {
		t.Fatalf("ProbeURL failed: %v", err)
	}
	if probeRes.Filename != "solid_data.bin" || probeRes.TotalSize != int64(len(payload)) {
		t.Fatalf("Unexpected probe result: %+v", probeRes)
	}

	// 2. Add and complete download
	dl, err := mgr.AddDownload(ctx, models.AddDownloadRequest{
		URL:         server.URL + "/myfile.bin",
		Filename:    "downloaded_solid.bin",
		SaveDir:     tempDir,
		Connections: 4,
	})
	if err != nil {
		t.Fatalf("AddDownload failed: %v", err)
	}

	deadline := time.Now().Add(5 * time.Second)
	var finalDL *models.Download
	for time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
		finalDL, err = repo.GetDownload(ctx, dl.ID)
		if err == nil && finalDL.Status == models.StatusCompleted {
			break
		}
	}

	if finalDL == nil || finalDL.Status != models.StatusCompleted {
		t.Fatalf("Download failed to complete. Status: %v, Error: %s", finalDL.Status, finalDL.ErrorMessage)
	}

	content, err := os.ReadFile(finalDL.SavePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if !bytes.Equal(content, payload) {
		t.Errorf("Downloaded payload content mismatch (len %d vs %d)", len(content), len(payload))
	}
}

func TestWorkerRangeIgnoredRejection(t *testing.T) {
	// Server ignores Range header and always returns 200 OK with full file
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("FULL_FILE_DATA"))
	}))
	defer server.Close()

	worker := NewChunkWorker(server.Client(), "TestAgent")
	chunk := &models.Chunk{
		ID:            "c1",
		ChunkIndex:    1,
		StartOffset:   100,
		EndOffset:     200,
		CurrentOffset: 100,
	}

	var totalCounter int64 = 0

	// WriterAt adapter for buffer
	writer := &testWriterAt{buf: make([]byte, 300)}
	err := worker.Download(context.Background(), server.URL, true, chunk, writer, &totalCounter, nil, nil)
	if err == nil {
		t.Fatalf("Expected error when server returns HTTP 200 for a range request at offset > 0, got nil")
	}
}

type testWriterAt struct {
	buf []byte
}

func (w *testWriterAt) WriteAt(p []byte, off int64) (n int, err error) {
	copy(w.buf[off:], p)
	return len(p), nil
}

func TestResumeUnfinishedDownloads(t *testing.T) {
	ctx := context.Background()
	payload := []byte("TEST_RESUME_UNFINISHED_DATA")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	tempDir, err := os.MkdirTemp("", "candak-resume-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	repo, err := sqlite.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to init sqlite repo: %v", err)
	}
	defer repo.Close()

	// Insert an interrupted download with StatusDownloading directly into repo
	dl := &models.Download{
		ID:             "dl_interrupted_1",
		URL:            server.URL + "/file.bin",
		Filename:       "interrupted.bin",
		SavePath:       filepath.Join(tempDir, "interrupted.bin"),
		TotalSize:      int64(len(payload)),
		DownloadedSize: 0,
		Status:         models.StatusDownloading,
		Resumable:      true,
		TotalChunks:    1,
		CreatedAt:      time.Now().Format(time.RFC3339),
		UpdatedAt:      time.Now().Format(time.RFC3339),
	}
	if err := repo.InsertDownload(ctx, dl); err != nil {
		t.Fatalf("Failed to save initial download: %v", err)
	}
	chunks := []*models.Chunk{
		{
			ID:            "chunk_1",
			DownloadID:    dl.ID,
			ChunkIndex:    0,
			StartOffset:   0,
			EndOffset:     int64(len(payload) - 1),
			CurrentOffset: 0,
			Status:        models.ChunkStatusDownloading,
		},
	}
	if err := repo.SaveChunks(ctx, chunks); err != nil {
		t.Fatalf("Failed to save initial chunks: %v", err)
	}

	mgr := NewManager(Config{
		Repository:   repo,
		Prober:       prober.NewHTTPProber(prober.WithHTTPClient(server.Client())),
		ChunkPlanner: planner.NewDefaultChunkPlanner(),
		Broadcaster:  events.NewNullBroadcaster(),
		HTTPClient:   server.Client(),
	})

	err = mgr.ResumeUnfinishedDownloads(ctx)
	if err != nil {
		t.Fatalf("ResumeUnfinishedDownloads failed: %v", err)
	}

	// Poll until completed
	deadline := time.Now().Add(5 * time.Second)
	var finalDL *models.Download
	for time.Now().Before(deadline) {
		finalDL, _ = repo.GetDownload(ctx, dl.ID)
		if finalDL != nil && finalDL.Status == models.StatusCompleted {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if finalDL == nil || finalDL.Status != models.StatusCompleted {
		t.Fatalf("Expected download to complete after resume, status: %v", finalDL.Status)
	}
}

func TestProgressThrottleCadence(t *testing.T) {
	mgr := NewManager(Config{
		Broadcaster: events.NewNullBroadcaster(),
	})
	defer mgr.Close()

	if !mgr.IsVisible() {
		t.Errorf("expected default visibility to be true")
	}
	if cadence := mgr.getProgressCadence(); cadence != 600*time.Millisecond {
		t.Errorf("expected visible cadence 600ms, got %v", cadence)
	}

	mgr.SetVisible(false)
	if mgr.IsVisible() {
		t.Errorf("expected visibility to be false after SetVisible(false)")
	}
	if cadence := mgr.getProgressCadence(); cadence != 2000*time.Millisecond {
		t.Errorf("expected minimized cadence 2000ms, got %v", cadence)
	}

	mgr.SetVisible(true)
	if cadence := mgr.getProgressCadence(); cadence != 600*time.Millisecond {
		t.Errorf("expected visible cadence 600ms, got %v", cadence)
	}
}

func TestBackgroundFlusherBatchOffsets(t *testing.T) {
	ctx := context.Background()
	tempDir, err := os.MkdirTemp("", "candak-flusher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	repo, err := sqlite.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to init sqlite repo: %v", err)
	}
	defer repo.Close()

	dl := &models.Download{
		ID:             "dl_flusher_test",
		URL:            "https://example.com/file.bin",
		Filename:       "file.bin",
		SavePath:       filepath.Join(tempDir, "file.bin"),
		TotalSize:      2000,
		DownloadedSize: 100,
		Status:         models.StatusDownloading,
		TotalChunks:    2,
	}
	if err := repo.InsertDownload(ctx, dl); err != nil {
		t.Fatalf("InsertDownload error: %v", err)
	}

	chunks := []*models.Chunk{
		{
			ID:            "ch_flush_0",
			DownloadID:    dl.ID,
			ChunkIndex:    0,
			StartOffset:   0,
			EndOffset:     999,
			CurrentOffset: 100,
			Status:        models.ChunkStatusDownloading,
		},
		{
			ID:            "ch_flush_1",
			DownloadID:    dl.ID,
			ChunkIndex:    1,
			StartOffset:   1000,
			EndOffset:     1999,
			CurrentOffset: 1000,
			Status:        models.ChunkStatusDownloading,
		},
	}
	if err := repo.SaveChunks(ctx, chunks); err != nil {
		t.Fatalf("SaveChunks error: %v", err)
	}

	mgr := NewManager(Config{
		Repository:  repo,
		Broadcaster: events.NewNullBroadcaster(),
	})
	defer mgr.Close()

	// Simulate active task
	taskCtx, taskCancel := context.WithCancel(context.Background())
	defer taskCancel()
	task := &activeTask{
		ctx:      taskCtx,
		cancel:   taskCancel,
		download: dl,
		chunks:   chunks,
		doneChan: make(chan struct{}),
	}
	mgr.mu.Lock()
	mgr.activeTasks[dl.ID] = task
	mgr.mu.Unlock()

	// Update chunk in-memory atomic counter
	chunks[0].CurrentOffset = 450
	chunks[1].CurrentOffset = 1350
	dl.DownloadedSize = 800

	// Flush dirty offsets across all active downloads
	mgr.flushDirtyOffsets(ctx)

	// Verify both chunks were updated in DB in single batch
	updatedChunks, err := repo.GetChunks(ctx, dl.ID)
	if err != nil {
		t.Fatalf("GetChunks failed: %v", err)
	}
	if updatedChunks[0].CurrentOffset != 450 || updatedChunks[1].CurrentOffset != 1350 {
		t.Errorf("chunk offsets not flushed properly: %+v", updatedChunks)
	}

	updatedDL, err := repo.GetDownload(ctx, dl.ID)
	if err != nil {
		t.Fatalf("GetDownload failed: %v", err)
	}
	if updatedDL.DownloadedSize != 800 {
		t.Errorf("download progress not flushed properly: %+v", updatedDL)
	}
}

func TestBatchProbeURLsAndAddBatchDownloads(t *testing.T) {
	ctx := context.Background()
	payload := []byte("BATCH_CONTENT_DATA_ABC")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	tempDir, err := os.MkdirTemp("", "candak-batch-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "batch_test.db")
	repo, err := sqlite.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to init sqlite repo: %v", err)
	}
	defer repo.Close()

	mgr := NewManager(Config{
		Repository:             repo,
		Prober:                 prober.NewHTTPProber(prober.WithHTTPClient(server.Client())),
		ChunkPlanner:           planner.NewDefaultChunkPlanner(),
		Broadcaster:            events.NewNullBroadcaster(),
		HTTPClient:             server.Client(),
		MaxConcurrentDownloads: 2,
	})

	urls := []string{
		server.URL + "/batch1.bin",
		server.URL + "/batch2.bin",
		server.URL + "/batch3.bin",
	}

	// 1. Test BatchProbeURLs
	probeResults, err := mgr.BatchProbeURLs(ctx, urls)
	if err != nil {
		t.Fatalf("BatchProbeURLs failed: %v", err)
	}
	if len(probeResults) != 3 {
		t.Fatalf("Expected 3 probe results, got %d", len(probeResults))
	}
	for i, pr := range probeResults {
		if pr.TotalSize != int64(len(payload)) {
			t.Errorf("Result %d total size mismatch: got %d, want %d", i, pr.TotalSize, len(payload))
		}
	}

	// 2. Test AddBatchDownloads with Concurrency Limit = 2
	reqs := []models.DownloadRequest{
		{URL: urls[0], Filename: "b1.bin", SaveDir: tempDir, Connections: 1, TotalSize: int64(len(payload)), Resumable: true},
		{URL: urls[1], Filename: "b2.bin", SaveDir: tempDir, Connections: 1, TotalSize: int64(len(payload)), Resumable: true},
		{URL: urls[2], Filename: "b3.bin", SaveDir: tempDir, Connections: 1, TotalSize: int64(len(payload)), Resumable: true},
	}

	ids, err := mgr.AddBatchDownloads(ctx, reqs)
	if err != nil {
		t.Fatalf("AddBatchDownloads failed: %v", err)
	}
	if len(ids) != 3 {
		t.Fatalf("Expected 3 created IDs, got %d", len(ids))
	}

	// Verify all downloads eventually complete via queue dispatching
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
		allDone := true
		for _, id := range ids {
			dl, err := repo.GetDownload(ctx, id)
			if err != nil || dl.Status != models.StatusCompleted {
				allDone = false
				break
			}
		}
		if allDone {
			break
		}
	}

	for _, id := range ids {
		dl, err := repo.GetDownload(ctx, id)
		if err != nil || dl.Status != models.StatusCompleted {
			t.Fatalf("Download %s did not complete: %+v", id, dl)
		}
		data, err := os.ReadFile(dl.SavePath)
		if err != nil || !bytes.Equal(data, payload) {
			t.Errorf("File %s content mismatch", dl.SavePath)
		}
	}

	// 3. Test Batch Collision Prevention (Same Filename)
	collisionReqs := []models.DownloadRequest{
		{URL: urls[0], Filename: "same.bin", SaveDir: tempDir, Connections: 4, Resumable: false},
		{URL: urls[1], Filename: "same.bin", SaveDir: tempDir, Connections: 4, Resumable: false},
	}
	collisionIDs, err := mgr.AddBatchDownloads(ctx, collisionReqs)
	if err != nil {
		t.Fatalf("AddBatchDownloads with duplicate names failed: %v", err)
	}
	dl1, _ := repo.GetDownload(ctx, collisionIDs[0])
	dl2, _ := repo.GetDownload(ctx, collisionIDs[1])
	if dl1.SavePath == dl2.SavePath {
		t.Errorf("Expected distinct save paths for collision items, both got %s", dl1.SavePath)
	}
	if dl1.TotalChunks != 1 || dl2.TotalChunks != 1 {
		t.Errorf("Non-resumable downloads should use 1 chunk connection, got %d and %d", dl1.TotalChunks, dl2.TotalChunks)
	}
}

func TestDynamicConcurrencyAndQueueControl(t *testing.T) {
	ctx := context.Background()
	payload := []byte("TEST_PAYLOAD_CONCURRENCY_CONTROL")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		time.Sleep(200 * time.Millisecond) // slow enough to stay active
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	tempDir, err := os.MkdirTemp("", "candak-concurrency-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	repo, err := sqlite.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to init sqlite repo: %v", err)
	}
	defer repo.Close()

	mgr := NewManager(Config{
		Repository:             repo,
		MaxConcurrentDownloads: 1,
	})
	defer mgr.Close()

	// 1. Clamping verification
	mgr.SetMaxConcurrentDownloads(0)
	if mgr.GetMaxConcurrentDownloads() != 1 {
		t.Errorf("Expected limit clamped to 1, got %d", mgr.GetMaxConcurrentDownloads())
	}
	mgr.SetMaxConcurrentDownloads(15)
	if mgr.GetMaxConcurrentDownloads() != 10 {
		t.Errorf("Expected limit clamped to 10, got %d", mgr.GetMaxConcurrentDownloads())
	}
	mgr.SetMaxConcurrentDownloads(1)

	// 2. Queue-aware manual resume under full queue
	// Simulate an active task occupying slot 1
	mgr.mu.Lock()
	mgr.activeTasks["active-slot-1"] = &activeTask{
		download: &models.Download{ID: "active-slot-1", Status: models.StatusDownloading},
	}
	mgr.mu.Unlock()

	// Create paused download in repo
	pausedDL := &models.Download{
		ID:             "paused-dl-1",
		URL:            server.URL + "/file.bin",
		Filename:       "file.bin",
		SavePath:       filepath.Join(tempDir, "file.bin"),
		TotalSize:      int64(len(payload)),
		DownloadedSize: 0,
		Status:         models.StatusPaused,
		Resumable:      false,
		TotalChunks:    1,
	}
	if err := repo.InsertDownload(ctx, pausedDL); err != nil {
		t.Fatalf("InsertDownload failed: %v", err)
	}
	chunk := &models.Chunk{
		ID:            "chunk_paused_0",
		DownloadID:    "paused-dl-1",
		ChunkIndex:    0,
		StartOffset:   0,
		EndOffset:     int64(len(payload) - 1),
		CurrentOffset: 0,
		Status:        models.ChunkStatusPending,
	}
	if err := repo.SaveChunks(ctx, []*models.Chunk{chunk}); err != nil {
		t.Fatalf("SaveChunks failed: %v", err)
	}

	// Resuming while slot is full should transition task into StatusQueued
	if err := mgr.ResumeDownload(ctx, "paused-dl-1"); err != nil {
		t.Fatalf("ResumeDownload failed: %v", err)
	}

	updatedDL, err := repo.GetDownload(ctx, "paused-dl-1")
	if err != nil {
		t.Fatalf("GetDownload failed: %v", err)
	}
	if updatedDL.Status != models.StatusQueued {
		t.Errorf("Expected resumed task to enter StatusQueued, got %s", updatedDL.Status)
	}

	// 3. Dynamic expansion: increasing concurrency limit dispatches queued task
	// Free active mock task
	mgr.mu.Lock()
	delete(mgr.activeTasks, "active-slot-1")
	mgr.mu.Unlock()

	mgr.SetMaxConcurrentDownloads(2)
	// Give dispatch routine a brief window
	time.Sleep(50 * time.Millisecond)

	mgr.mu.Lock()
	activeLen := len(mgr.activeTasks)
	mgr.mu.Unlock()
	if activeLen != 1 {
		t.Errorf("Expected queued task to be dispatched on limit increase, active tasks: %d", activeLen)
	}

	// 4. Batch resume race prevention: rapid calls respect concurrency ceiling
	mgr.SetMaxConcurrentDownloads(1)
	paused2 := &models.Download{
		ID:       "paused-dl-2",
		URL:      server.URL + "/file2.bin",
		Filename: "file2.bin",
		SavePath: filepath.Join(tempDir, "file2.bin"),
		Status:   models.StatusPaused,
	}
	paused3 := &models.Download{
		ID:       "paused-dl-3",
		URL:      server.URL + "/file3.bin",
		Filename: "file3.bin",
		SavePath: filepath.Join(tempDir, "file3.bin"),
		Status:   models.StatusPaused,
	}
	_ = repo.InsertDownload(ctx, paused2)
	_ = repo.InsertDownload(ctx, paused3)
	_ = repo.SaveChunks(ctx, []*models.Chunk{{ID: "c2", DownloadID: "paused-dl-2", CurrentOffset: 0, EndOffset: 10}})
	_ = repo.SaveChunks(ctx, []*models.Chunk{{ID: "c3", DownloadID: "paused-dl-3", CurrentOffset: 0, EndOffset: 10}})

	// Currently 1 task is running (paused-dl-1). Rapidly resume both paused2 and paused3
	_ = mgr.ResumeDownload(ctx, "paused-dl-2")
	_ = mgr.ResumeDownload(ctx, "paused-dl-3")

	d2, _ := repo.GetDownload(ctx, "paused-dl-2")
	d3, _ := repo.GetDownload(ctx, "paused-dl-3")
	if d2.Status != models.StatusQueued || d3.Status != models.StatusQueued {
		t.Errorf("Expected both batch-resumed items to be queued, got d2=%s, d3=%s", d2.Status, d3.Status)
	}
}
