package downloader

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"candak/internal/events"
	"candak/internal/models"
	"candak/internal/planner"
	"candak/internal/prober"
	"candak/internal/storage"
	"candak/internal/sysutil"
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
	notifier   sysutil.Notifier

	visible atomic.Bool

	maxConcurrentDownloads int

	mu          sync.RWMutex
	activeTasks map[string]*activeTask

	flusherStop chan struct{}
	flusherDone chan struct{}
}

// Config provides configuration parameters for Manager.
type Config struct {
	Repository             storage.Repository
	Prober                 prober.URLProber
	ChunkPlanner           planner.ChunkPlanner
	Broadcaster            events.Broadcaster
	Notifier               sysutil.Notifier
	HTTPClient             *http.Client
	UserAgent              string
	MaxConcurrentDownloads int
}

// NewManager constructs a download Manager with injected dependencies.
func NewManager(cfg Config) *Manager {
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{
			Timeout: 0,
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

	ua := cfg.UserAgent
	if ua == "" {
		ua = "Candak-Download-Manager/0.2.0"
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

	maxConcurrent := cfg.MaxConcurrentDownloads
	if maxConcurrent <= 0 {
		maxConcurrent = 3
	}

	m := &Manager{
		repo:                   cfg.Repository,
		prober:                 pr,
		planner:                pl,
		emitter:                em,
		httpClient:             client,
		worker:                 NewChunkWorker(client, ua),
		notifier:               cfg.Notifier,
		maxConcurrentDownloads: maxConcurrent,
		activeTasks:            make(map[string]*activeTask),
		flusherStop:            make(chan struct{}),
		flusherDone:            make(chan struct{}),
	}
	m.visible.Store(true)

	go m.runBackgroundFlusher()

	return m
}

// SetVisible dynamically adjusts the progress broadcast frequency based on window visibility.
func (m *Manager) SetVisible(visible bool) {
	m.visible.Store(visible)
}

// IsVisible returns whether the application window is currently visible.
func (m *Manager) IsVisible() bool {
	return m.visible.Load()
}

func (m *Manager) getProgressCadence() time.Duration {
	if m.IsVisible() {
		return 600 * time.Millisecond
	}
	return 2000 * time.Millisecond
}

// Close cleanly terminates background flusher tasks.
func (m *Manager) Close() {
	select {
	case <-m.flusherStop:
		return
	default:
		close(m.flusherStop)
		<-m.flusherDone
	}
}

func (m *Manager) runBackgroundFlusher() {
	defer close(m.flusherDone)
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.flusherStop:
			m.flushDirtyOffsets(context.Background())
			return
		case <-ticker.C:
			m.flushDirtyOffsets(context.Background())
		}
	}
}

func (m *Manager) flushDirtyOffsets(ctx context.Context) {
	m.mu.RLock()
	if len(m.activeTasks) == 0 {
		m.mu.RUnlock()
		return
	}

	var chunkUpdates []storage.ChunkOffsetUpdate
	type progressRecord struct {
		id             string
		downloadedSize int64
		speed          int64
		eta            int64
		status         models.DownloadStatus
	}
	var progressRecords []progressRecord

	for _, task := range m.activeTasks {
		if task.download.Status == models.StatusDownloading {
			for _, c := range task.chunks {
				offset := atomic.LoadInt64(&c.CurrentOffset)
				chunkUpdates = append(chunkUpdates, storage.ChunkOffsetUpdate{
					ChunkID:       c.ID,
					CurrentOffset: offset,
					Status:        c.Status,
				})
			}
			progressRecords = append(progressRecords, progressRecord{
				id:             task.download.ID,
				downloadedSize: task.download.DownloadedSize,
				speed:          task.download.Speed,
				eta:            task.download.ETA,
				status:         task.download.Status,
			})
		}
	}
	m.mu.RUnlock()

	if len(chunkUpdates) > 0 {
		_ = m.repo.BatchUpdateChunkOffsets(ctx, chunkUpdates)
	}
	for _, pr := range progressRecords {
		_ = m.repo.UpdateDownloadProgress(ctx, pr.id, pr.downloadedSize, pr.speed, pr.eta, pr.status)
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

// ProbeURLWithHeaders analyzes target metadata with custom headers via the injected URLProber.
func (m *Manager) ProbeURLWithHeaders(ctx context.Context, rawURL string, headers map[string]string) (*models.ProbeResult, error) {
	return m.prober.ProbeWithHeaders(ctx, rawURL, headers)
}

// AddDownload initializes, stores, and starts a download task.
func (m *Manager) AddDownload(ctx context.Context, req models.AddDownloadRequest) (*models.Download, error) {
	if req.URL == "" {
		return nil, models.ErrInvalidURL
	}

	probeRes, err := m.prober.ProbeWithHeaders(ctx, req.URL, req.Headers)
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
		Headers:        req.Headers,
		Chunks:         chunks,
	}

	if err := m.repo.InsertDownload(ctx, dl); err != nil {
		return nil, fmt.Errorf("failed to insert download: %w", err)
	}
	if err := m.repo.SaveChunks(ctx, chunks); err != nil {
		return nil, fmt.Errorf("failed to save chunks: %w", err)
	}
	_ = m.repo.RecordURLHistory(ctx, req.URL, filename)

	m.mu.Lock()
	if len(m.activeTasks) < m.maxConcurrentDownloads {
		m.mu.Unlock()
		go m.startTask(dl, chunks)
	} else {
		m.mu.Unlock()
		m.emitStatusChange(dl)
	}

	return dl, nil
}

// BatchProbeURLs concurrently inspects multiple target URLs via worker pool.
func (m *Manager) BatchProbeURLs(ctx context.Context, urls []string) ([]models.ProbeResult, error) {
	return prober.BatchProbe(ctx, m.prober, urls, 4)
}

// AddBatchDownloads enqueues multiple downloads and dispatches active tasks up to the concurrency limit.
func (m *Manager) AddBatchDownloads(ctx context.Context, requests []models.DownloadRequest) ([]string, error) {
	if len(requests) == 0 {
		return []string{}, nil
	}

	var createdIDs []string
	home, _ := os.UserHomeDir()
	defaultSaveDir := filepath.Join(home, "Downloads")
	reservedPaths := make(map[string]bool)

	for _, req := range requests {
		trimmedURL := strings.TrimSpace(req.URL)
		if trimmedURL == "" {
			continue
		}

		saveDir := req.SaveDir
		if saveDir == "" {
			saveDir = defaultSaveDir
		}

		filename := req.Filename
		if filename == "" {
			if parsed, err := url.Parse(trimmedURL); err == nil {
				filename = filepath.Base(parsed.Path)
			}
			if filename == "" || filename == "." || filename == "/" {
				filename = "download.bin"
			}
		}
		filename = prober.SanitizeFilename(filename)
		savePath := ensureUniqueFilePath(filepath.Join(saveDir, filename), reservedPaths)
		filename = filepath.Base(savePath)

		conns := req.Connections
		if conns <= 0 {
			conns = 4
		}
		if !req.Resumable {
			conns = 1
		}

		dlID := generateID()
		chunks := m.planner.Plan(dlID, req.TotalSize, req.Resumable, conns)

		dl := &models.Download{
			ID:             dlID,
			URL:            trimmedURL,
			Filename:       filename,
			SavePath:       savePath,
			TotalSize:      req.TotalSize,
			DownloadedSize: 0,
			Status:         models.StatusQueued,
			Resumable:      req.Resumable,
			TotalChunks:    len(chunks),
			Speed:          0,
			ETA:            0,
			ErrorMessage:   "",
			Headers:        req.Headers,
			Chunks:         chunks,
		}

		if err := m.repo.InsertDownload(ctx, dl); err != nil {
			continue
		}
		if err := m.repo.SaveChunks(ctx, chunks); err != nil {
			continue
		}
		_ = m.repo.RecordURLHistory(ctx, trimmedURL, filename)

		// Emit status change immediately for real-time dashboard rendering
		m.emitStatusChange(dl)
		createdIDs = append(createdIDs, dlID)
	}

	// Dispatch active tasks according to the queue limit
	m.dispatchNextQueued()

	return createdIDs, nil
}

func (m *Manager) dispatchNextQueued() {
	m.mu.Lock()
	defer m.mu.Unlock()

	activeCount := len(m.activeTasks)
	available := m.maxConcurrentDownloads - activeCount
	if available <= 0 {
		return
	}

	downloads, err := m.repo.GetAllDownloads(context.Background())
	if err != nil {
		return
	}

	var queued []*models.Download
	for _, d := range downloads {
		if d.Status == models.StatusQueued {
			if _, active := m.activeTasks[d.ID]; !active {
				queued = append(queued, d)
			}
		}
	}

	sort.Slice(queued, func(i, j int) bool {
		return queued[i].CreatedAt < queued[j].CreatedAt
	})

	toStart := available
	if len(queued) < toStart {
		toStart = len(queued)
	}

	for i := 0; i < toStart; i++ {
		dl := queued[i]
		chunks, err := m.repo.GetChunks(context.Background(), dl.ID)
		if err != nil || len(chunks) == 0 {
			continue
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
		go m.runTask(task)
	}
}

func ensureUniqueFilePath(path string, reserved ...map[string]bool) string {
	isAvailable := func(p string) bool {
		if len(reserved) > 0 && reserved[0] != nil && reserved[0][p] {
			return false
		}
		_, err := os.Stat(p)
		return os.IsNotExist(err)
	}

	if isAvailable(path) {
		if len(reserved) > 0 && reserved[0] != nil {
			reserved[0][path] = true
		}
		return path
	}

	dir := filepath.Dir(path)
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(filepath.Base(path), ext)

	for i := 1; i < 1000; i++ {
		candidate := filepath.Join(dir, fmt.Sprintf("%s (%d)%s", base, i, ext))
		if isAvailable(candidate) {
			if len(reserved) > 0 && reserved[0] != nil {
				reserved[0][candidate] = true
			}
			return candidate
		}
	}
	if len(reserved) > 0 && reserved[0] != nil {
		reserved[0][path] = true
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

	m.runTask(task)
}

func (m *Manager) runTask(task *activeTask) {
	dl := task.download
	chunks := task.chunks
	ctx := task.ctx

	defer func() {
		m.mu.Lock()
		delete(m.activeTasks, dl.ID)
		m.mu.Unlock()
		close(task.doneChan)
		go m.dispatchNextQueued()
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

	var currentDownloadedBytes int64 = 0
	for _, c := range chunks {
		if c.CurrentOffset > c.StartOffset {
			currentDownloadedBytes += (c.CurrentOffset - c.StartOffset)
		}
	}
	if currentDownloadedBytes == 0 && dl.DownloadedSize > 0 {
		currentDownloadedBytes = dl.DownloadedSize
	}
	dl.DownloadedSize = currentDownloadedBytes
	metrics := NewMetricsCalculator(currentDownloadedBytes)

	// Progress loop timer (600ms visible, 2000ms minimized)
	progressTimer := time.NewTimer(m.getProgressCadence())
	defer progressTimer.Stop()

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
			if err := m.worker.Download(ctx, dl.URL, dl.Resumable, c, file, &currentDownloadedBytes, persistCallback, dl.Headers); err != nil {
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
			case now := <-progressTimer.C:
				total := atomic.LoadInt64(&currentDownloadedBytes)
				dl.DownloadedSize = total
				dl.Speed, dl.ETA = metrics.Update(now, total, dl.TotalSize)

				m.emitProgress(dl, chunks)
				progressTimer.Reset(m.getProgressCadence())
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
	finalSize := atomic.LoadInt64(&currentDownloadedBytes)
	if dl.TotalSize > 0 {
		finalSize = dl.TotalSize
	}
	dl.DownloadedSize = finalSize
	dl.Speed = 0
	dl.ETA = 0
	_ = m.repo.UpdateDownloadProgress(context.Background(), dl.ID, finalSize, 0, 0, models.StatusCompleted)
	_ = m.repo.UpdateDownloadStatus(context.Background(), dl.ID, models.StatusCompleted, "")
	for _, c := range chunks {
		c.Status = models.ChunkStatusCompleted
	}
	m.emitProgress(dl, chunks)
	m.emitStatusChange(dl)
	if m.notifier != nil {
		_ = m.notifier.Notify("Candak Download Complete", fmt.Sprintf("Finished: %s", dl.Filename))
	}
}

func (m *Manager) finishWithError(dl *models.Download, errMsg string) {
	dl.Status = models.StatusError
	dl.ErrorMessage = errMsg
	dl.Speed = 0
	dl.ETA = 0
	_ = m.repo.UpdateDownloadStatus(context.Background(), dl.ID, models.StatusError, errMsg)
	m.emitStatusChange(dl)
	if m.notifier != nil {
		_ = m.notifier.Notify("Candak Download Failed", fmt.Sprintf("Failed: %s - %s", dl.Filename, errMsg))
	}
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

// ResumeUnfinishedDownloads finds tasks in downloading, queued, or pending states from previous session and resumes them.
func (m *Manager) ResumeUnfinishedDownloads(ctx context.Context) error {
	downloads, err := m.repo.GetAllDownloads(ctx)
	if err != nil {
		return err
	}

	for _, dl := range downloads {
		if dl.Status == models.StatusDownloading || dl.Status == models.StatusQueued || dl.Status == "pending" {
			_ = m.ResumeDownload(ctx, dl.ID)
		}
	}
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

// PauseAll cleanly suspends all active download tasks.
func (m *Manager) PauseAll(ctx context.Context) {
	m.mu.Lock()
	tasks := make([]*activeTask, 0, len(m.activeTasks))
	for _, t := range m.activeTasks {
		t.download.Status = models.StatusPaused
		t.cancel()
		tasks = append(tasks, t)
	}
	m.mu.Unlock()

	for _, t := range tasks {
		select {
		case <-t.doneChan:
		case <-time.After(1 * time.Second):
		}
		_ = m.repo.UpdateDownloadStatus(ctx, t.download.ID, models.StatusPaused, "")
	}
}

// GetAllDownloads returns all registered downloads, synchronizing any inactive tasks.
func (m *Manager) GetAllDownloads(ctx context.Context) ([]*models.Download, error) {
	downloads, err := m.repo.GetAllDownloads(ctx)
	if err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, dl := range downloads {
		if dl.Status == models.StatusDownloading || dl.Status == models.StatusQueued {
			if _, active := m.activeTasks[dl.ID]; !active {
				dl.Status = models.StatusPaused
				dl.Speed = 0
				dl.ETA = 0
				_ = m.repo.UpdateDownloadProgress(ctx, dl.ID, dl.DownloadedSize, 0, 0, models.StatusPaused)
				_ = m.repo.UpdateDownloadStatus(ctx, dl.ID, models.StatusPaused, "")
			}
		}
	}
	return downloads, nil
}

// GetURLHistory returns the URL history list.
func (m *Manager) GetURLHistory(ctx context.Context, limit int) ([]*models.URLHistoryItem, error) {
	return m.repo.GetURLHistory(ctx, limit)
}

// DeleteURLHistoryItem removes a specific entry from the URL history.
func (m *Manager) DeleteURLHistoryItem(ctx context.Context, id int64) error {
	return m.repo.DeleteURLHistoryItem(ctx, id)
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
