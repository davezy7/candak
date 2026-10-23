package storage

import (
	"context"

	"candak/internal/models"
)

// DownloadRepository defines read and write operations for downloads.
type DownloadRepository interface {
	// InsertDownload records a new download entity.
	InsertDownload(ctx context.Context, dl *models.Download) error
	// UpdateDownloadProgress updates byte counter, transfer speed, and ETA.
	UpdateDownloadProgress(ctx context.Context, id string, downloadedSize int64, speed int64, eta int64, status models.DownloadStatus) error
	// UpdateDownloadStatus updates the download's operational status and error text.
	UpdateDownloadStatus(ctx context.Context, id string, status models.DownloadStatus, errMsg string) error
	// GetDownload retrieves a single download by ID, including its associated chunks.
	GetDownload(ctx context.Context, id string) (*models.Download, error)
	// GetAllDownloads retrieves all downloads ordered chronologically descending.
	GetAllDownloads(ctx context.Context) ([]*models.Download, error)
	// DeleteDownload removes a download record and cascades chunk deletion.
	DeleteDownload(ctx context.Context, id string) error
}

// ChunkRepository defines storage operations for chunk partitions.
type ChunkRepository interface {
	// SaveChunks transactionally inserts or updates a slice of chunk records.
	SaveChunks(ctx context.Context, chunks []*models.Chunk) error
	// UpdateChunkOffset persists a chunk's progress offset and status.
	UpdateChunkOffset(ctx context.Context, chunkID string, currentOffset int64, status models.ChunkStatus) error
	// GetChunks retrieves all chunks assigned to a download ID in ascending index order.
	GetChunks(ctx context.Context, downloadID string) ([]*models.Chunk, error)
}

// HistoryRepository defines storage operations for target URL history.
type HistoryRepository interface {
	// RecordURLHistory inserts or increments the usage of a URL.
	RecordURLHistory(ctx context.Context, urlStr, filename string) error
	// GetURLHistory retrieves the most recently used URLs up to the specified limit.
	GetURLHistory(ctx context.Context, limit int) ([]*models.URLHistoryItem, error)
	// DeleteURLHistoryItem removes a specific URL record from history by ID.
	DeleteURLHistoryItem(ctx context.Context, id int64) error
	// ClearURLHistory purges all recorded URL history records.
	ClearURLHistory(ctx context.Context) error
}

// Repository aggregates Download, Chunk, and History repositories.
type Repository interface {
	DownloadRepository
	ChunkRepository
	HistoryRepository
	// Close cleanly terminates the storage connection.
	Close() error
}
