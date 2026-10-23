package downloader

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"candak/internal/events"
	"candak/internal/models"
	"candak/internal/planner"
	"candak/internal/prober"
	"candak/internal/storage"
)

type activeTask struct {
	ctx      context.Context
	cancel   context.CancelFunc
	download *models.Download
	chunks   []*models.Chunk
	file     *os.File
	doneChan chan struct{}
}

// Manager orchestrates multi-chunk download tasks and state persistence.
type Manager struct {
	repo       storage.Repository
	prober     prober.URLProber
	planner    planner.ChunkPlanner
	emitter    events.Broadcaster
	httpClient *http.Client
	worker     *ChunkWorker

	mu          sync.RWMutex
	activeTasks map[string]*activeTask
}

// Config provides configuration parameters for Manager.
type Config struct {
	Repository   storage.Repository
	Prober       prober.URLProber
	ChunkPlanner planner.ChunkPlanner
	Broadcaster  events.Broadcaster
	HTTPClient   *http.Client
	UserAgent    string
}

// NewManager constructs a download Manager with injected dependencies.
func NewManager(cfg Config) *Manager {
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{
			Timeout: 0,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 20,
				IdleConnTimeout:     90 * time.Second,
			},
		}
	}

	ua := cfg.UserAgent
	if ua == "" {
		ua = "Candak-Download-Manager/1.0"
	}

	pr := cfg.Prober
	if pr == nil {
		pr = prober.NewHTTPProber(prober.WithHTTPClient(client), prober.WithUserAgent(ua))
	}

	pl := cfg.ChunkPlanner
	if pl == nil {
		pl = planner.NewDefaultChunkPlanner()
	}

	em := cfg.Broadcaster
	if em == nil {
		em = events.NewNullBroadcaster()
	}

	return &Manager{
		repo:        cfg.Repository,
		prober:      pr,
		planner:     pl,
		emitter:     em,
		httpClient:  client,
		worker:      NewChunkWorker(client, ua),
		activeTasks: make(map[string]*activeTask),
	}
}

// SetBroadcaster updates the event broadcaster (e.g. after Wails context is ready).
func (m *Manager) SetBroadcaster(b events.Broadcaster) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.emitter = b
}

func generateID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ProbeURL analyzes target metadata via the injected URLProber.
func (m *Manager) ProbeURL(ctx context.Context, rawURL string) (*models.ProbeResult, error) {
	return m.prober.Probe(ctx, rawURL)
}

// AddDownload initializes, stores, and starts a download task.
func (m *Manager) AddDownload(ctx context.Context, req models.AddDownloadRequest) (*models.Download, error) {
	if req.URL == "" {
		return nil, models.ErrInvalidURL
	}

	probeRes, err := m.prober.Probe(ctx, req.URL)
	if err != nil {
		return nil, err
	}

	filename := req.Filename
	if filename == "" {
		filename = probeRes.Filename
	}
	filename = prober.SanitizeFilename(filename)

	saveDir := req.SaveDir
	if saveDir == "" {
		home, _ := os.UserHomeDir()
		saveDir = filepath.Join(home, "Downloads")
	}

	savePath := ensureUniqueFilePath(filepath.Join(saveDir, filename))
	filename = filepath.Base(savePath)

	dlID := generateID()
	chunks := m.planner.Plan(dlID, probeRes.TotalSize, probeRes.Resumable, req.Connections)

	dl := &models.Download{
		ID:             dlID,
		URL:            req.URL,
		Filename:       filename,
		SavePath:       savePath,
		TotalSize:      probeRes.TotalSize,
		DownloadedSize: 0,
		Status:         models.StatusQueued,
		Resumable:      probeRes.Resumable,
		TotalChunks:    len(chunks),
		Speed:          0,
		ETA:            0,
		ErrorMessage:   "",
		Chunks:         chunks,
	}

	if err := m.repo.InsertDownload(ctx, dl); err != nil {
		return nil, fmt.Errorf("failed to insert download: %w", err)
	}
	if err := m.repo.SaveChunks(ctx, chunks); err != nil {
		return nil, fmt.Errorf("failed to save chunks: %w", err)
	}
	_ = m.repo.RecordURLHistory(ctx, req.URL, filename)

	// Launch download worker goroutine
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
	task := &activeTask{
		ctx:      ctx,
		cancel:   cancel,
		download: dl,
		chunks:   chunks,
		doneChan: make(chan struct{}),
	}
	m.activeTasks[dl.ID] = task
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		delete(m.activeTasks, dl.ID)
		m.mu.Unlock()
		close(task.doneChan)
	}()

	// Ensure destination file
	file, err := os.OpenFile(dl.SavePath, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		m.finishWithError(dl, fmt.Sprintf("%v: %v", models.ErrDestinationAccess, err))
		return
	}
	defer file.Close()
	task.file = file

	if dl.TotalSize > 0 {
		_ = file.Truncate(dl.TotalSize)
	}

	dl.Status = models.StatusDownloading
	_ = m.repo.UpdateDownloadStatus(ctx, dl.ID, models.StatusDownloading, "")
	m.emitStatusChange(dl)

	var currentDownloadedBytes int64 = dl.DownloadedSize
	metrics := NewMetricsCalculator(currentDownloadedBytes)

	// Progress loop ticker
	progressTicker := time.NewTicker(300 * time.Millisecond)
	defer progressTicker.Stop()

	var wg sync.WaitGroup
	errChan := make(chan error, len(chunks))

	persistCallback := func(cCtx context.Context, chunkID string, currentOffset int64, status models.ChunkStatus) error {
		return m.repo.UpdateChunkOffset(cCtx, chunkID, currentOffset, status)
	}

	for _, chunk := range chunks {
		if chunk.CurrentOffset > chunk.EndOffset && chunk.EndOffset > 0 {
			chunk.Status = models.ChunkStatusCompleted
			continue
		}

		wg.Add(1)
		go func(c *models.Chunk) {
			defer wg.Done()
			if err := m.worker.Download(ctx, dl.URL, dl.Resumable, c, file, &currentDownloadedBytes, persistCallback); err != nil {
				if ctx.Err() == nil {
					errChan <- fmt.Errorf("chunk %d: %w", c.ChunkIndex, err)
				}
			}
		}(chunk)
	}

	tickerDone := make(chan struct{})
	go func() {
		for {
			select {
			case <-task.ctx.Done():
				return
			case <-tickerDone:
				return
			case now := <-progressTicker.C:
				total := atomic.LoadInt64(&currentDownloadedBytes)
				dl.DownloadedSize = total
				dl.Speed, dl.ETA = metrics.Update(now, total, dl.TotalSize)

				_ = m.repo.UpdateDownloadProgress(ctx, dl.ID, dl.DownloadedSize, dl.Speed, dl.ETA, dl.Status)
				m.emitProgress(dl, chunks)
			}
		}
	}()

	wg.Wait()
	close(tickerDone)

	// Check if cancelled or paused
	if ctx.Err() != nil {
		_ = m.repo.UpdateDownloadProgress(context.Background(), dl.ID, dl.DownloadedSize, 0, 0, dl.Status)
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

	// Mark completed
	dl.Status = models.StatusCompleted
	dl.DownloadedSize = dl.TotalSize
	dl.Speed = 0
	dl.ETA = 0
	_ = m.repo.UpdateDownloadProgress(context.Background(), dl.ID, dl.TotalSize, 0, 0, models.StatusCompleted)
	_ = m.repo.UpdateDownloadStatus(context.Background(), dl.ID, models.StatusCompleted, "")
	for _, c := range chunks {
		c.Status = models.ChunkStatusCompleted
	}
	m.emitProgress(dl, chunks)
	m.emitStatusChange(dl)
}

func (m *Manager) finishWithError(dl *models.Download, errMsg string) {
	dl.Status = models.StatusError
	dl.ErrorMessage = errMsg
	dl.Speed = 0
	dl.ETA = 0
	_ = m.repo.UpdateDownloadStatus(context.Background(), dl.ID, models.StatusError, errMsg)
	m.emitStatusChange(dl)
}

// PauseDownload suspends an active download and preserves offset progress.
func (m *Manager) PauseDownload(ctx context.Context, id string) error {
	m.mu.Lock()
	task, exists := m.activeTasks[id]
	if !exists {
		m.mu.Unlock()
		dl, err := m.repo.GetDownload(ctx, id)
		if err != nil {
			return err
		}
		if dl.Status == models.StatusDownloading || dl.Status == models.StatusQueued {
			return m.repo.UpdateDownloadStatus(ctx, id, models.StatusPaused, "")
		}
		return nil
	}

	task.download.Status = models.StatusPaused
	task.cancel()
	m.mu.Unlock()

	<-task.doneChan
	_ = m.repo.UpdateDownloadStatus(ctx, id, models.StatusPaused, "")
	dl, _ := m.repo.GetDownload(ctx, id)
	if dl != nil {
		m.emitStatusChange(dl)
	}
	return nil
}

// ResumeDownload restarts a paused download from saved chunk offsets.
func (m *Manager) ResumeDownload(ctx context.Context, id string) error {
	m.mu.Lock()
	if _, exists := m.activeTasks[id]; exists {
		m.mu.Unlock()
		return models.ErrAlreadyRunning
	}
	m.mu.Unlock()

	dl, err := m.repo.GetDownload(ctx, id)
	if err != nil {
		return err
	}
	if dl.Status == models.StatusCompleted {
		return models.ErrAlreadyCompleted
	}

	chunks, err := m.repo.GetChunks(ctx, id)
	if err != nil {
		return err
	}

	go m.startTask(dl, chunks)
	return nil
}

// CancelDownload cancels a download and optionally purges the file from disk.
func (m *Manager) CancelDownload(ctx context.Context, id string, deleteFile bool) error {
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

	dl, err := m.repo.GetDownload(ctx, id)
	if err != nil && err != models.ErrDownloadNotFound {
		return err
	}

	if deleteFile && dl != nil && dl.SavePath != "" {
		_ = os.Remove(dl.SavePath)
	}

	if err := m.repo.DeleteDownload(ctx, id); err != nil {
		return err
	}

	m.emitter.Emit("download:deleted", id)
	return nil
}

// GetAllDownloads returns all registered downloads.
func (m *Manager) GetAllDownloads(ctx context.Context) ([]*models.Download, error) {
	return m.repo.GetAllDownloads(ctx)
}

// GetURLHistory returns the URL history list.
func (m *Manager) GetURLHistory(ctx context.Context, limit int) ([]*models.URLHistoryItem, error) {
	return m.repo.GetURLHistory(ctx, limit)
}

// ClearURLHistory purges the URL history records.
func (m *Manager) ClearURLHistory(ctx context.Context) error {
	return m.repo.ClearURLHistory(ctx)
}

func (m *Manager) emitProgress(dl *models.Download, chunks []*models.Chunk) {
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
	m.emitter.Emit("download:progress", evt)
}

func (m *Manager) emitStatusChange(dl *models.Download) {
	m.emitter.Emit("download:status_change", dl)
}
