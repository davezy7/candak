package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"

	"candak/internal/models"
)

// ChunkOffsetPersister defines a callback function to persist updated chunk offsets.
type ChunkOffsetPersister func(ctx context.Context, chunkID string, currentOffset int64, status models.ChunkStatus) error

// ChunkWorker executes byte range retrieval for a single chunk.
type ChunkWorker struct {
	client    *http.Client
	userAgent string
}

// NewChunkWorker constructs a ChunkWorker.
func NewChunkWorker(client *http.Client, userAgent string) *ChunkWorker {
	if client == nil {
		client = http.DefaultClient
	}
	if userAgent == "" {
		userAgent = "Candak-Download-Manager/1.0"
	}
	return &ChunkWorker{
		client:    client,
		userAgent: userAgent,
	}
}

// Download processes a chunk's remaining byte range, writing directly into the target WriterAt.
func (w *ChunkWorker) Download(
	ctx context.Context,
	url string,
	resumable bool,
	chunk *models.Chunk,
	writer io.WriterAt,
	totalCounter *int64,
	persist ChunkOffsetPersister,
) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", w.userAgent)

	rangeRequested := resumable && chunk.EndOffset > 0
	if rangeRequested {
		rangeHeader := fmt.Sprintf("bytes=%d-%d", chunk.CurrentOffset, chunk.EndOffset)
		req.Header.Set("Range", rangeHeader)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if rangeRequested && chunk.CurrentOffset > 0 && resp.StatusCode == http.StatusOK {
		return fmt.Errorf("chunk %d requested range from offset %d, but server returned HTTP 200 (full body)", chunk.ChunkIndex, chunk.CurrentOffset)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("unexpected HTTP response status %d for chunk %d", resp.StatusCode, chunk.ChunkIndex)
	}

	chunk.Status = models.ChunkStatusDownloading

	// Always guarantee the latest byte offset is persisted upon exit
	defer func() {
		if persist != nil {
			status := chunk.Status
			if ctx.Err() != nil {
				status = models.ChunkStatusPending
			}
			_ = persist(context.Background(), chunk.ID, chunk.CurrentOffset, status)
		}
	}()

	buf := make([]byte, 64*1024) // 64 KB buffer
	lastPersisted := time.Now()

	var bodyReader io.Reader = resp.Body
	if chunk.EndOffset > 0 && chunk.EndOffset >= chunk.CurrentOffset {
		remaining := chunk.EndOffset - chunk.CurrentOffset + 1
		bodyReader = io.LimitReader(resp.Body, remaining)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, readErr := bodyReader.Read(buf)
		if n > 0 {
			_, writeErr := writer.WriteAt(buf[:n], chunk.CurrentOffset)
			if writeErr != nil {
				chunk.Status = models.ChunkStatusError
				return fmt.Errorf("chunk %d file write error: %w", chunk.ChunkIndex, writeErr)
			}

			chunk.CurrentOffset += int64(n)
			atomic.AddInt64(totalCounter, int64(n))

			// Periodically save chunk offset to database (every 1 second)
			if time.Since(lastPersisted) > 1*time.Second {
				if persist != nil {
					_ = persist(ctx, chunk.ID, chunk.CurrentOffset, models.ChunkStatusDownloading)
				}
				lastPersisted = time.Now()
			}
		}

		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			chunk.Status = models.ChunkStatusError
			return readErr
		}
	}

	chunk.Status = models.ChunkStatusCompleted
	return nil
}
