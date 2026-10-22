package downloader

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"candak/internal/db"
	"candak/internal/models"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type activeDownload struct {
	ctx        context.Context
	cancel     context.CancelFunc
	download   *models.Download
	chunks     []*models.Chunk
	file       *os.File
	mu         sync.Mutex
	lastBytes  int64
	lastTime   time.Time
	speedEMA   float64
	doneChan   chan struct{}
}

type Manager struct {
	db          *db.Database
	appCtx      context.Context
	httpClient  *http.Client
	mu          sync.RWMutex
	activeTasks map[string]*activeDownload
}

func NewManager(database *db.Database) *Manager {
	return &Manager{
		db: database,
		httpClient: &http.Client{
			Timeout: 0, // No client-level timeout for large file downloads
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 20,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		activeTasks: make(map[string]*activeDownload),
	}
}

func (m *Manager) SetAppContext(ctx context.Context) {
	m.appCtx = ctx
}

func generateID() string {
	bytes := make([]byte, 8)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (m *Manager) ProbeURL(rawURL string) (*models.ProbeResult, error) {
	parsedURL, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	req, err := http.NewRequest("HEAD", parsedURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Candak-Download-Manager/1.0 (Mozilla/5.0)")

	resp, err := m.httpClient.Do(req)
	// If HEAD is rejected or fails, attempt GET with Range: bytes=0-0
	if err != nil || resp.StatusCode >= 400 {
		getReq, err := http.NewRequest("GET", parsedURL.String(), nil)
		if err != nil {
			return nil, err
		}
		getReq.Header.Set("User-Agent", "Candak-Download-Manager/1.0 (Mozilla/5.0)")
		getReq.Header.Set("Range", "bytes=0-0")
		resp, err = m.httpClient.Do(getReq)
		if err != nil {
			return nil, fmt.Errorf("failed to probe URL: %w", err)
		}
		defer resp.Body.Close()
	} else {
		defer resp.Body.Close()
	}

	totalSize := resp.ContentLength
	resumable := false

	if resp.Header.Get("Accept-Ranges") == "bytes" || resp.Header.Get("Content-Range") != "" {
		resumable = true
	}

	// Try extracting filename
	filename := ""
	contentDisp := resp.Header.Get("Content-Disposition")
	if contentDisp != "" {
		if _, params, err := mime.ParseMediaType(contentDisp); err == nil {
			if fn, ok := params["filename*"]; ok {
				parts := strings.SplitN(fn, "''", 2)
				if len(parts) == 2 {
					if unesc, err := url.PathUnescape(parts[1]); err == nil {
						filename = unesc
					}
				}
			}
			if filename == "" {
				filename = params["filename"]
			}
		}
	}

	if filename == "" {
		urlPath := parsedURL.Path
		filename = filepath.Base(urlPath)
		if filename == "." || filename == "/" || filename == "" {
			filename = "download.bin"
		} else {
			if unesc, err := url.PathUnescape(filename); err == nil {
				filename = unesc
			}
		}
	}

	filename = sanitizeFilename(filename)

	return &models.ProbeResult{
		URL:         rawURL,
		Filename:    filename,
		TotalSize:   totalSize,
		Resumable:   resumable,
		ContentType: resp.Header.Get("Content-Type"),
	}, nil
}

func sanitizeFilename(name string) string {
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	clean := name
	for _, c := range invalidChars {
		clean = strings.ReplaceAll(clean, c, "_")
	}
	clean = strings.TrimSpace(clean)
	if clean == "" {
		clean = "download.bin"
	}
	return clean
}

func (m *Manager) AddDownload(req models.AddDownloadRequest) (*models.Download, error) {
	if req.URL == "" {
		return nil, fmt.Errorf("URL cannot be empty")
	}

	probe, err := m.ProbeURL(req.URL)
	if err != nil {
		return nil, err
	}

	filename := req.Filename
	if filename == "" {
		filename = probe.Filename
	}
	filename = sanitizeFilename(filename)

	saveDir := req.SaveDir
	if saveDir == "" {
		home, _ := os.UserHomeDir()
		saveDir = filepath.Join(home, "Downloads")
	}

	savePath := filepath.Join(saveDir, filename)
	// If file exists, add a suffix
	savePath = ensureUniqueFilePath(savePath)
	filename = filepath.Base(savePath)

	connections := req.Connections
	if connections <= 0 {
		connections = 4
	}
	if !probe.Resumable || probe.TotalSize <= 0 {
		connections = 1
	}

	dlID := generateID()
	dl := &models.Download{
		ID:             dlID,
		URL:            req.URL,
		Filename:       filename,
		SavePath:       savePath,
		TotalSize:      probe.TotalSize,
		DownloadedSize: 0,
		Status:         models.StatusQueued,
		Resumable:      probe.Resumable,
		TotalChunks:    connections,
		Speed:          0,
		ETA:            0,
		ErrorMessage:   "",
	}

	// Create chunks
	var chunks []*models.Chunk
	if probe.Resumable && probe.TotalSize > 0 && connections > 1 {
		chunkSize := probe.TotalSize / int64(connections)
		for i := 0; i < connections; i++ {
			start := int64(i) * chunkSize
			end := start + chunkSize - 1
			if i == connections-1 {
				end = probe.TotalSize - 1
			}
			chunks = append(chunks, &models.Chunk{
				ID:            fmt.Sprintf("%s_c%d", dlID, i),
				DownloadID:    dlID,
				ChunkIndex:    i,
				StartOffset:   start,
				EndOffset:     end,
				CurrentOffset: start,
				Status:        models.ChunkStatusPending,
			})
		}
	} else {
		endOffset := probe.TotalSize - 1
		if endOffset < 0 {
			endOffset = 0
		}
		chunks = append(chunks, &models.Chunk{
			ID:            fmt.Sprintf("%s_c0", dlID),
			DownloadID:    dlID,
			ChunkIndex:    0,
			StartOffset:   0,
			EndOffset:     endOffset,
			CurrentOffset: 0,
			Status:        models.ChunkStatusPending,
		})
	}

	dl.Chunks = chunks

	if err := m.db.InsertDownload(dl); err != nil {
		return nil, fmt.Errorf("failed to save download to database: %w", err)
	}
	if err := m.db.SaveChunks(chunks); err != nil {
		return nil, fmt.Errorf("failed to save chunks to database: %w", err)
	}
	_ = m.db.RecordURLHistory(req.URL, filename)

	// Start the download immediately
	go m.startTask(dl, chunks)

	return dl, nil
}

func ensureUniqueFilePath(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}
	dir := filepath.Dir(path)
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(filepath.Base(path), ext)

	for i := 1; i < 1000; i++ {
		candidate := filepath.Join(dir, fmt.Sprintf("%s (%d)%s", base, i, ext))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
	return path
}

func (m *Manager) startTask(dl *models.Download, chunks []*models.Chunk) {
	m.mu.Lock()
	if _, exists := m.activeTasks[dl.ID]; exists {
		m.mu.Unlock()
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	task := &activeDownload{
		ctx:        ctx,
		cancel:     cancel,
		download:   dl,
		chunks:     chunks,
		doneChan:   make(chan struct{}),
		lastBytes:  dl.DownloadedSize,
		lastTime:   time.Now(),
	}
	m.activeTasks[dl.ID] = task
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		delete(m.activeTasks, dl.ID)
		m.mu.Unlock()
		close(task.doneChan)
	}()

	// Open or create file for writing chunks at offset
	file, err := os.OpenFile(dl.SavePath, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		m.finishWithError(dl, fmt.Sprintf("failed to create destination file: %v", err))
		return
	}
	defer file.Close()
	task.file = file

	if dl.TotalSize > 0 {
		_ = file.Truncate(dl.TotalSize)
	}

	dl.Status = models.StatusDownloading
	_ = m.db.UpdateDownloadStatus(dl.ID, models.StatusDownloading, "")
	m.emitStatusChange(dl)

	// Start ticker for progress & speed calculation
	progressTicker := time.NewTicker(300 * time.Millisecond)
	defer progressTicker.Stop()

	var wg sync.WaitGroup
	errChan := make(chan error, len(chunks))

	var currentDownloadedBytes int64 = dl.DownloadedSize

	for _, chunk := range chunks {
		if chunk.CurrentOffset > chunk.EndOffset && chunk.EndOffset > 0 {
			chunk.Status = models.ChunkStatusCompleted
			continue
		}

		wg.Add(1)
		go func(c *models.Chunk) {
			defer wg.Done()
			if err := m.downloadChunk(ctx, dl, c, file, &currentDownloadedBytes); err != nil {
				if ctx.Err() == nil {
					errChan <- fmt.Errorf("chunk %d failed: %w", c.ChunkIndex, err)
				}
			}
		}(chunk)
	}

	// Progress loop
	tickerDone := make(chan struct{})
	go func() {
		for {
			select {
			case <-task.ctx.Done():
				return
			case <-tickerDone:
				return
			case now := <-progressTicker.C:
				totalDownloaded := atomic.LoadInt64(&currentDownloadedBytes)
				dl.DownloadedSize = totalDownloaded

				// Compute speed
				timeDelta := now.Sub(task.lastTime).Seconds()
				if timeDelta > 0 {
					bytesDelta := totalDownloaded - task.lastBytes
					if bytesDelta < 0 {
						bytesDelta = 0
					}
					instantSpeed := float64(bytesDelta) / timeDelta
					if task.speedEMA == 0 {
						task.speedEMA = instantSpeed
					} else {
						task.speedEMA = (0.7 * instantSpeed) + (0.3 * task.speedEMA)
					}
					dl.Speed = int64(task.speedEMA)
					if dl.Speed > 0 && dl.TotalSize > totalDownloaded {
						dl.ETA = (dl.TotalSize - totalDownloaded) / dl.Speed
					} else {
						dl.ETA = 0
					}
					task.lastBytes = totalDownloaded
					task.lastTime = now
				}

				_ = m.db.UpdateDownloadProgress(dl.ID, dl.DownloadedSize, dl.Speed, dl.ETA, dl.Status)
				m.emitProgress(dl, chunks)
			}
		}
	}()

	wg.Wait()
	close(tickerDone)

	// Check if cancelled or paused
	if ctx.Err() != nil {
		_ = m.db.UpdateDownloadProgress(dl.ID, dl.DownloadedSize, 0, 0, dl.Status)
		m.emitStatusChange(dl)
		return
	}

	// Check chunk errors
	select {
	case err := <-errChan:
		m.finishWithError(dl, err.Error())
		return
	default:
	}

	// Completed
	dl.Status = models.StatusCompleted
	dl.DownloadedSize = dl.TotalSize
	dl.Speed = 0
	dl.ETA = 0
	_ = m.db.UpdateDownloadProgress(dl.ID, dl.TotalSize, 0, 0, models.StatusCompleted)
	_ = m.db.UpdateDownloadStatus(dl.ID, models.StatusCompleted, "")
	for _, c := range chunks {
		c.Status = models.ChunkStatusCompleted
	}
	m.emitProgress(dl, chunks)
	m.emitStatusChange(dl)
}

func (m *Manager) downloadChunk(ctx context.Context, dl *models.Download, c *models.Chunk, file *os.File, totalDownloaded *int64) error {
	req, err := http.NewRequestWithContext(ctx, "GET", dl.URL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Candak-Download-Manager/1.0 (Mozilla/5.0)")

	// If ranges are supported and not entire file without end offset
	if dl.Resumable && c.EndOffset > 0 {
		rangeHeader := fmt.Sprintf("bytes=%d-%d", c.CurrentOffset, c.EndOffset)
		req.Header.Set("Range", rangeHeader)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("unexpected HTTP status: %d", resp.StatusCode)
	}

	c.Status = models.ChunkStatusDownloading

	buf := make([]byte, 64*1024)
	lastPersisted := time.Now()

	for {
		select {
		case <-ctx.Done():
			_ = m.db.UpdateChunkOffset(c.ID, c.CurrentOffset, models.ChunkStatusPending)
			return ctx.Err()
		default:
		}

		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := file.WriteAt(buf[:n], c.CurrentOffset)
			if writeErr != nil {
				return fmt.Errorf("file write error: %w", writeErr)
			}
			c.CurrentOffset += int64(n)
			atomic.AddInt64(totalDownloaded, int64(n))

			// Persist chunk offset to DB every 1 second
			if time.Since(lastPersisted) > 1*time.Second {
				_ = m.db.UpdateChunkOffset(c.ID, c.CurrentOffset, models.ChunkStatusDownloading)
				lastPersisted = time.Now()
			}
		}

		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return readErr
		}
	}

	c.Status = models.ChunkStatusCompleted
	_ = m.db.UpdateChunkOffset(c.ID, c.CurrentOffset, models.ChunkStatusCompleted)
	return nil
}

func (m *Manager) finishWithError(dl *models.Download, errMsg string) {
	dl.Status = models.StatusError
	dl.ErrorMessage = errMsg
	dl.Speed = 0
	dl.ETA = 0
	_ = m.db.UpdateDownloadStatus(dl.ID, models.StatusError, errMsg)
	m.emitStatusChange(dl)
}

func (m *Manager) PauseDownload(id string) error {
	m.mu.Lock()
	task, exists := m.activeTasks[id]
	if !exists {
		m.mu.Unlock()
		// If not in active tasks, check DB and mark paused if needed
		dl, err := m.db.GetDownload(id)
		if err != nil {
			return err
		}
		if dl.Status == models.StatusDownloading || dl.Status == models.StatusQueued {
			return m.db.UpdateDownloadStatus(id, models.StatusPaused, "")
		}
		return nil
	}
	task.download.Status = models.StatusPaused
	task.cancel()
	m.mu.Unlock()

	<-task.doneChan
	_ = m.db.UpdateDownloadStatus(id, models.StatusPaused, "")
	dl, _ := m.db.GetDownload(id)
	if dl != nil {
		m.emitStatusChange(dl)
	}
	return nil
}

func (m *Manager) ResumeDownload(id string) error {
	m.mu.Lock()
	if _, exists := m.activeTasks[id]; exists {
		m.mu.Unlock()
		return fmt.Errorf("download is already running")
	}
	m.mu.Unlock()

	dl, err := m.db.GetDownload(id)
	if err != nil {
		return err
	}
	if dl.Status == models.StatusCompleted {
		return fmt.Errorf("download is already completed")
	}

	chunks, err := m.db.GetChunks(id)
	if err != nil {
		return err
	}

	go m.startTask(dl, chunks)
	return nil
}

func (m *Manager) CancelDownload(id string, deleteFile bool) error {
	m.mu.Lock()
	task, exists := m.activeTasks[id]
	if exists {
		task.download.Status = models.StatusCancelled
		task.cancel()
		m.mu.Unlock()
		<-task.doneChan
	} else {
		m.mu.Unlock()
	}

	dl, err := m.db.GetDownload(id)
	if err != nil {
		return err
	}

	if deleteFile && dl.SavePath != "" {
		_ = os.Remove(dl.SavePath)
	}

	if err := m.db.DeleteDownload(id); err != nil {
		return err
	}

	if m.appCtx != nil {
		wailsRuntime.EventsEmit(m.appCtx, "download:deleted", id)
	}
	return nil
}

func (m *Manager) GetAllDownloads() ([]*models.Download, error) {
	return m.db.GetAllDownloads()
}

func (m *Manager) GetURLHistory(limit int) ([]*models.URLHistoryItem, error) {
	return m.db.GetURLHistory(limit)
}

func (m *Manager) ClearURLHistory() error {
	return m.db.ClearURLHistory()
}

func (m *Manager) emitProgress(dl *models.Download, chunks []*models.Chunk) {
	if m.appCtx == nil {
		return
	}
	var progress float64 = 0
	if dl.TotalSize > 0 {
		progress = (float64(dl.DownloadedSize) / float64(dl.TotalSize)) * 100.0
		if progress > 100 {
			progress = 100
		}
	}

	evt := models.ProgressEvent{
		DownloadID:     dl.ID,
		DownloadedSize: dl.DownloadedSize,
		TotalSize:      dl.TotalSize,
		Progress:       progress,
		Speed:          dl.Speed,
		ETA:            dl.ETA,
		Status:         dl.Status,
		Chunks:         chunks,
	}
	wailsRuntime.EventsEmit(m.appCtx, "download:progress", evt)
}

func (m *Manager) emitStatusChange(dl *models.Download) {
	if m.appCtx == nil {
		return
	}
	wailsRuntime.EventsEmit(m.appCtx, "download:status_change", dl)
}
