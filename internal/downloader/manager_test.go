package downloader

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"candak/internal/db"
	"candak/internal/models"
)

func TestDownloaderManager(t *testing.T) {
	// Create mock payload (200 KB)
	payload := bytes.Repeat([]byte("ABCDEFGHIJ0123456789"), 10240) // 200 KB

	// Create test HTTP server with Accept-Ranges support
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Disposition", "attachment; filename=\"testdata.bin\"")

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

		// Handle Range: bytes=start-end
		if !strings.HasPrefix(rangeHeader, "bytes=") {
			w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
			return
		}

		rangeSpec := strings.TrimPrefix(rangeHeader, "bytes=")
		parts := strings.Split(rangeSpec, "-")
		start, _ := strconv.ParseInt(parts[0], 10, 64)
		end := int64(len(payload) - 1)
		if len(parts) > 1 && parts[1] != "" {
			end, _ = strconv.ParseInt(parts[1], 10, 64)
		}

		if start > end || start >= int64(len(payload)) {
			w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
			return
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

	// Initialize temp test DB
	tempDir, err := os.MkdirTemp("", "candak-downloader-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "downloader_test.db")
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to init db: %v", err)
	}
	defer database.Close()

	mgr := NewManager(database)

	// 1. Test ProbeURL
	probe, err := mgr.ProbeURL(server.URL + "/myfile.bin")
	if err != nil {
		t.Fatalf("ProbeURL failed: %v", err)
	}
	if !probe.Resumable {
		t.Errorf("Expected resumable=true, got false")
	}
	if probe.TotalSize != int64(len(payload)) {
		t.Errorf("Expected TotalSize=%d, got %d", len(payload), probe.TotalSize)
	}
	if probe.Filename != "testdata.bin" {
		t.Errorf("Expected Filename=testdata.bin, got %s", probe.Filename)
	}

	// 2. Test AddDownload with 4 concurrent chunks
	dl, err := mgr.AddDownload(models.AddDownloadRequest{
		URL:         server.URL + "/myfile.bin",
		Filename:    "downloaded_test.bin",
		SaveDir:     tempDir,
		Connections: 4,
	})
	if err != nil {
		t.Fatalf("AddDownload failed: %v", err)
	}

	// Wait for download completion (up to 5s)
	deadline := time.Now().Add(5 * time.Second)
	var finalDL *models.Download
	for time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
		finalDL, err = database.GetDownload(dl.ID)
		if err == nil && finalDL.Status == models.StatusCompleted {
			break
		}
	}

	if finalDL == nil || finalDL.Status != models.StatusCompleted {
		t.Fatalf("Download failed to complete in time. Status: %v, Error: %s", finalDL.Status, finalDL.ErrorMessage)
	}

	// Verify file on disk
	content, err := os.ReadFile(finalDL.SavePath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}
	if !bytes.Equal(content, payload) {
		t.Errorf("Downloaded file content does not match original payload (size %d vs %d)", len(content), len(payload))
	}

	// Verify chunks status in DB
	chunks, err := database.GetChunks(dl.ID)
	if err != nil {
		t.Fatalf("Failed to get chunks from DB: %v", err)
	}
	if len(chunks) != 4 {
		t.Errorf("Expected 4 chunks, got %d", len(chunks))
	}
	for _, c := range chunks {
		if c.Status != models.ChunkStatusCompleted {
			t.Errorf("Chunk %d status not completed: %s", c.ChunkIndex, c.Status)
		}
	}
}
