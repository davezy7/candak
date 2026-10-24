package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"candak/internal/models"
	"candak/internal/storage"
)

func TestSQLiteStorage(t *testing.T) {
	ctx := context.Background()
	tempDir, err := os.MkdirTemp("", "candak-sqlite-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	store, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to init sqlite store: %v", err)
	}
	defer store.Close()

	// 1. Insert & Retrieve
	dl := &models.Download{
		ID:             "dl_solid_1",
		URL:            "https://example.com/solid.iso",
		Filename:       "solid.iso",
		SavePath:       filepath.Join(tempDir, "solid.iso"),
		TotalSize:      50000000,
		DownloadedSize: 0,
		Status:         models.StatusQueued,
		Resumable:      true,
		TotalChunks:    2,
	}

	if err := store.InsertDownload(ctx, dl); err != nil {
		t.Fatalf("InsertDownload error: %v", err)
	}

	got, err := store.GetDownload(ctx, dl.ID)
	if err != nil {
		t.Fatalf("GetDownload error: %v", err)
	}
	if got.Filename != dl.Filename || got.TotalSize != dl.TotalSize {
		t.Errorf("Mismatch in download: got %+v, want %+v", got, dl)
	}

	// 2. Chunks & Offsets
	chunks := []*models.Chunk{
		{
			ID:            "chunk_1_0",
			DownloadID:    dl.ID,
			ChunkIndex:    0,
			StartOffset:   0,
			EndOffset:     24999999,
			CurrentOffset: 0,
			Status:        models.ChunkStatusPending,
		},
		{
			ID:            "chunk_1_1",
			DownloadID:    dl.ID,
			ChunkIndex:    1,
			StartOffset:   25000000,
			EndOffset:     49999999,
			CurrentOffset: 25000000,
			Status:        models.ChunkStatusPending,
		},
	}
	if err := store.SaveChunks(ctx, chunks); err != nil {
		t.Fatalf("SaveChunks error: %v", err)
	}

	if err := store.UpdateChunkOffset(ctx, "chunk_1_0", 123456, models.ChunkStatusDownloading); err != nil {
		t.Fatalf("UpdateChunkOffset error: %v", err)
	}

	chunksFromDB, err := store.GetChunks(ctx, dl.ID)
	if err != nil {
		t.Fatalf("GetChunks error: %v", err)
	}
	if len(chunksFromDB) != 2 || chunksFromDB[0].CurrentOffset != 123456 {
		t.Errorf("Unexpected chunks: %+v", chunksFromDB)
	}

	// Test BatchUpdateChunkOffsets in single transaction
	batchUpdates := []storage.ChunkOffsetUpdate{
		{ChunkID: "chunk_1_0", CurrentOffset: 500000, Status: models.ChunkStatusDownloading},
		{ChunkID: "chunk_1_1", CurrentOffset: 25500000, Status: models.ChunkStatusCompleted},
	}
	if err := store.BatchUpdateChunkOffsets(ctx, batchUpdates); err != nil {
		t.Fatalf("BatchUpdateChunkOffsets error: %v", err)
	}
	updatedChunks, err := store.GetChunks(ctx, dl.ID)
	if err != nil {
		t.Fatalf("GetChunks after batch update error: %v", err)
	}
	if updatedChunks[0].CurrentOffset != 500000 || updatedChunks[1].CurrentOffset != 25500000 || updatedChunks[1].Status != models.ChunkStatusCompleted {
		t.Errorf("Unexpected chunks after batch update: %+v", updatedChunks)
	}

	// Verify batched chunks attachment in GetAllDownloads
	allDownloads, err := store.GetAllDownloads(ctx)
	if err != nil {
		t.Fatalf("GetAllDownloads error: %v", err)
	}
	if len(allDownloads) != 1 || len(allDownloads[0].Chunks) != 2 {
		t.Fatalf("Expected 1 download with 2 batched chunks, got %+v", allDownloads)
	}

	// 3. URL History
	if err := store.RecordURLHistory(ctx, dl.URL, dl.Filename); err != nil {
		t.Fatalf("RecordURLHistory error: %v", err)
	}
	history, err := store.GetURLHistory(ctx, 10)
	if err != nil || len(history) != 1 {
		t.Fatalf("GetURLHistory unexpected: %+v (err: %v)", history, err)
	}
	if history[0].URL != dl.URL {
		t.Errorf("Expected URL %s, got %s", dl.URL, history[0].URL)
	}

	if err := store.DeleteURLHistoryItem(ctx, history[0].ID); err != nil {
		t.Fatalf("DeleteURLHistoryItem error: %v", err)
	}
	remaining, err := store.GetURLHistory(ctx, 10)
	if err != nil || len(remaining) != 0 {
		t.Fatalf("Expected 0 history items after deletion, got %+v", remaining)
	}
}

func TestInterruptedDownloadsReconciliation(t *testing.T) {
	ctx := context.Background()
	tempDir, err := os.MkdirTemp("", "candak-reconcile-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	store, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to init sqlite store: %v", err)
	}

	dl := &models.Download{
		ID:             "dl_interrupted",
		URL:            "https://example.com/file.bin",
		Filename:       "file.bin",
		SavePath:       filepath.Join(tempDir, "file.bin"),
		TotalSize:      1000,
		DownloadedSize: 500,
		Status:         models.StatusDownloading,
		Speed:          50000,
		ETA:            10,
	}
	if err := store.InsertDownload(ctx, dl); err != nil {
		t.Fatalf("InsertDownload error: %v", err)
	}
	_ = store.Close()

	// Reopen store (simulating app restart after unexpected exit)
	store2, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to reopen sqlite store: %v", err)
	}
	defer store2.Close()

	reconciled, err := store2.GetDownload(ctx, dl.ID)
	if err != nil {
		t.Fatalf("GetDownload error: %v", err)
	}
	if reconciled.Status != models.StatusPaused {
		t.Errorf("Expected status to be paused on restart, got %v", reconciled.Status)
	}
	if reconciled.Speed != 0 || reconciled.ETA != 0 {
		t.Errorf("Expected speed and ETA to be reset to 0, got speed=%d, eta=%d", reconciled.Speed, reconciled.ETA)
	}
}
