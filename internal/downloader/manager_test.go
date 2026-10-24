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
