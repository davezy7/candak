package downloader

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"candak/internal/models"
)

const workerBufferSize = 256 * 1024 // 256 KB buffer for high-throughput reads

var bufferPool = sync.Pool{
	New: func() any {
		b := make([]byte, workerBufferSize)
		return &b
	},
}

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
		client = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				ForceAttemptHTTP2:   true,
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 32,
				IdleConnTimeout:     90 * time.Second,
				TLSHandshakeTimeout: 10 * time.Second,
				DisableCompression:  true,
				WriteBufferSize:     128 * 1024,
				ReadBufferSize:      128 * 1024,
			},
		}
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
	headers map[string]string,
) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", w.userAgent)
	for k, v := range headers {
		if strings.EqualFold(k, "Range") {
			continue
		}
		req.Header.Set(k, v)
	}

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
			_ = persist(context.Background(), chunk.ID, atomic.LoadInt64(&chunk.CurrentOffset), status)
		}
	}()

	bufPtr := bufferPool.Get().(*[]byte)
	defer bufferPool.Put(bufPtr)
	buf := *bufPtr

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
			current := atomic.LoadInt64(&chunk.CurrentOffset)
			_, writeErr := writer.WriteAt(buf[:n], current)
			if writeErr != nil {
				chunk.Status = models.ChunkStatusError
				return fmt.Errorf("chunk %d file write error: %w", chunk.ChunkIndex, writeErr)
			}

			atomic.AddInt64(&chunk.CurrentOffset, int64(n))
			atomic.AddInt64(totalCounter, int64(n))
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
