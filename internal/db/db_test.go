package db

import (
	"os"
	"path/filepath"
	"testing"

	"candak/internal/models"
)

func TestDatabaseOperations(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "candak-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	database, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to init db: %v", err)
	}
	defer database.Close()

	// 1. Test Download Insertion & Retrieval
	dl := &models.Download{
		ID:             "dl_test_1",
		URL:            "https://example.com/file.iso",
		Filename:       "file.iso",
		SavePath:       "/tmp/file.iso",
		TotalSize:      104857600, // 100MB
		DownloadedSize: 0,
		Status:         models.StatusQueued,
		Resumable:      true,
		TotalChunks:    4,
		Speed:          0,
		ETA:            0,
	}

	if err := database.InsertDownload(dl); err != nil {
		t.Fatalf("InsertDownload failed: %v", err)
	}

	retrieved, err := database.GetDownload(dl.ID)
	if err != nil {
		t.Fatalf("GetDownload failed: %v", err)
	}
	if retrieved.Filename != dl.Filename || retrieved.TotalSize != dl.TotalSize {
		t.Errorf("Retrieved download mismatch: %+v", retrieved)
	}

	// 2. Test Chunks Tracking & Offsets (Resume Support)
	chunks := []*models.Chunk{
		{
			ID:            "dl_test_1_c0",
			DownloadID:    dl.ID,
			ChunkIndex:    0,
			StartOffset:   0,
			EndOffset:     26214399,
			CurrentOffset: 0,
			Status:        models.ChunkStatusPending,
		},
		{
			ID:            "dl_test_1_c1",
			DownloadID:    dl.ID,
			ChunkIndex:    1,
			StartOffset:   26214400,
			EndOffset:     52428799,
			CurrentOffset: 26214400,
			Status:        models.ChunkStatusPending,
		},
	}

	if err := database.SaveChunks(chunks); err != nil {
		t.Fatalf("SaveChunks failed: %v", err)
	}

	// Simulate progress on Chunk 0: advanced offset to 10MB
	newOffset := int64(10485760)
	if err := database.UpdateChunkOffset("dl_test_1_c0", newOffset, models.ChunkStatusDownloading); err != nil {
		t.Fatalf("UpdateChunkOffset failed: %v", err)
	}

	loadedChunks, err := database.GetChunks(dl.ID)
	if err != nil {
		t.Fatalf("GetChunks failed: %v", err)
	}
	if len(loadedChunks) != 2 {
		t.Fatalf("Expected 2 chunks, got %d", len(loadedChunks))
	}
	if loadedChunks[0].CurrentOffset != newOffset {
		t.Errorf("Expected current offset %d, got %d", newOffset, loadedChunks[0].CurrentOffset)
	}
	if loadedChunks[0].Status != models.ChunkStatusDownloading {
		t.Errorf("Expected chunk status downloading, got %s", loadedChunks[0].Status)
	}

	// 3. Test URL History Tracking
	if err := database.RecordURLHistory(dl.URL, dl.Filename); err != nil {
		t.Fatalf("RecordURLHistory failed: %v", err)
	}
	// Record same URL again to increment count
	if err := database.RecordURLHistory(dl.URL, dl.Filename); err != nil {
		t.Fatalf("RecordURLHistory duplicate failed: %v", err)
	}

	history, err := database.GetURLHistory(10)
	if err != nil {
		t.Fatalf("GetURLHistory failed: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("Expected 1 history item, got %d", len(history))
	}
	if history[0].DownloadCount != 2 {
		t.Errorf("Expected download count 2, got %d", history[0].DownloadCount)
	}

	// 4. Test Delete
	if err := database.DeleteDownload(dl.ID); err != nil {
		t.Fatalf("DeleteDownload failed: %v", err)
	}
	remainingChunks, _ := database.GetChunks(dl.ID)
	if len(remainingChunks) != 0 {
		t.Errorf("Expected 0 cascaded chunks after deletion, got %d", len(remainingChunks))
	}
}
