package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"candak/internal/clipboard"
	"candak/internal/downloader"
	"candak/internal/events"
	"candak/internal/models"
	"candak/internal/planner"
	"candak/internal/prober"
	"candak/internal/server"
	"candak/internal/storage"
	"candak/internal/storage/sqlite"
	"candak/internal/sysutil"
	"candak/internal/tray"
	"candak/internal/updater"
	"candak/internal/ytdl"
	"sync"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:browser-extension
var embeddedExtension embed.FS

type ytTask struct {
	cancel    context.CancelFunc
	rawURL    string
	formatID  string
	saveDir   string
	filename  string
	outPath   string
	totalSize int64
}

// App acts as the boundary adapter between the Wails frontend IPC and domain services.
type App struct {
	ctx              context.Context
	repo             storage.Repository
	manager          *downloader.Manager
	broadcaster      *events.WailsBroadcaster
	opener           sysutil.SystemOpener
	notifier         sysutil.Notifier
	autostart        sysutil.Autostarter
	loopbackServer   *server.LoopbackServer
	clipboardWatcher *clipboard.Watcher
	tray             tray.Tray
	mu               sync.Mutex
	isPromptOnly     bool
	isMinimized      bool
	ytMu             sync.Mutex
	ytTasks          map[string]*ytTask
}

// NewApp constructs a new App application struct.
func NewApp() *App {
	broadcaster := events.NewWailsBroadcaster()
	app := &App{
		broadcaster: broadcaster,
		opener:      sysutil.NewOSOpener(),
		notifier:    sysutil.NewOSNotifier(),
		autostart:   sysutil.NewAutostart(),
		ytTasks:     make(map[string]*ytTask),
	}
	app.tray = tray.NewTray(tray.Callbacks{
		OnOpen: func() {
			app.OpenMainWindow()
		},
		OnPauseAll: func() {
			if app.manager != nil && app.ctx != nil {
				app.manager.PauseAll(app.ctx)
			}
		},
		OnQuit: func() {
			if app.ctx != nil {
				wailsRuntime.Quit(app.ctx)
			}
		},
	})
	return app
}

// startup is called by Wails when the desktop application launches.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.broadcaster.SetContext(ctx)

	// Determine user storage path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	appDir := filepath.Join(homeDir, ".candak")
	_ = os.MkdirAll(appDir, 0755)
	_ = a.ensureExtensionFiles(filepath.Join(appDir, "browser-extension"))

	dbPath := filepath.Join(appDir, "candak.db")
	repo, err := sqlite.New(dbPath)
	if err != nil {
		// Fallback to current working directory if user home is not writable
		fallbackAppDir := ".candak"
		_ = os.MkdirAll(fallbackAppDir, 0755)
		fallbackDbPath := filepath.Join(fallbackAppDir, "candak.db")
		repo, err = sqlite.New(fallbackDbPath)
		if err != nil {
			wailsRuntime.LogErrorf(ctx, "Failed to initialize SQLite storage: %v", err)
			return
		}
	}
	a.repo = repo

	// Compose Download Manager with injected dependencies (SOLID DIP)
	a.manager = downloader.NewManager(downloader.Config{
		Repository:   repo,
		Prober:       prober.NewHTTPProber(),
		ChunkPlanner: planner.NewDefaultChunkPlanner(),
		Broadcaster:  a.broadcaster,
		Notifier:     a.notifier,
	})

	// Start Loopback Server for Browser Integration
	a.loopbackServer = server.NewLoopbackServer(server.Config{
		Port: server.DefaultPort,
		OnGrab: func(req models.GrabRequest) error {
			a.HandleGrabRequest(req)
			return nil
		},
	})
	if err := a.loopbackServer.Start(); err != nil {
		wailsRuntime.LogWarningf(ctx, "Failed to start loopback grabber server: %v", err)
	}

	// Initialize Clipboard Watcher
	a.clipboardWatcher = clipboard.NewWatcher(clipboard.Config{
		Getter: func() (string, error) {
			return wailsRuntime.ClipboardGetText(a.ctx)
		},
		OnDetect: func(detectedURL string) {
			a.HandleGrabRequest(models.GrabRequest{
				URL: detectedURL,
			})
		},
		Interval: 1 * time.Second,
	})

	// Initialize System Tray
	if a.tray != nil {
		if err := a.tray.Start(); err != nil {
			wailsRuntime.LogWarningf(ctx, "Failed to initialize system tray: %v", err)
		}
	}

	// On boot in Tray Daemon Mode (--minimized), automatically resume uncompleted downloads in the background
	if a.isMinimized {
		if a.manager != nil {
			a.manager.SetVisible(false)
		}
		go func() {
			if a.manager != nil {
				_ = a.manager.ResumeUnfinishedDownloads(a.ctx)
			}
		}()
	}
}

// shutdown is called by Wails when the desktop application terminates.
func (a *App) shutdown(ctx context.Context) {
	if a.tray != nil {
		a.tray.Stop()
	}
	if a.manager != nil {
		a.manager.PauseAll(ctx)
		a.manager.Close()
	}
	if a.loopbackServer != nil {
		_ = a.loopbackServer.Stop(ctx)
	}
	if a.clipboardWatcher != nil {
		a.clipboardWatcher.Stop()
	}
	if a.repo != nil {
		_ = a.repo.Close()
	}
}

// ProbeURL analyzes target URL metadata and range capabilities.
func (a *App) ProbeURL(rawURL string) (*models.ProbeResult, error) {
	if a.manager == nil {
		return nil, fmt.Errorf("download manager is not ready")
	}
	return a.manager.ProbeURL(a.ctx, rawURL)
}

// BatchProbeURLs concurrently analyzes multiple URLs using a worker pool.
func (a *App) BatchProbeURLs(urls []string) ([]models.ProbeResult, error) {
	if a.manager == nil {
		return nil, fmt.Errorf("download manager is not ready")
	}
	return a.manager.BatchProbeURLs(a.ctx, urls)
}

// AddDownload initializes, stores, and starts a download task.
func (a *App) AddDownload(req models.AddDownloadRequest) (*models.Download, error) {
	if a.manager == nil {
		return nil, fmt.Errorf("download manager is not ready")
	}
	return a.manager.AddDownload(a.ctx, req)
}

// AddBatchDownloads enqueues multiple downloads and dispatches active tasks according to concurrency limits.
func (a *App) AddBatchDownloads(requests []models.DownloadRequest) ([]string, error) {
	if a.manager == nil {
		return nil, fmt.Errorf("download manager is not ready")
	}
	return a.manager.AddBatchDownloads(a.ctx, requests)
}

// PauseDownload suspends an active download task and persists chunk offsets.
func (a *App) PauseDownload(id string) error {
	if strings.HasPrefix(id, "yt_") {
		return a.pauseYouTubeDownload(id)
	}
	if a.manager == nil {
		return fmt.Errorf("download manager is not ready")
	}
	return a.manager.PauseDownload(a.ctx, id)
}

// ResumeDownload restarts a paused download using saved chunk offsets.
func (a *App) ResumeDownload(id string) error {
	if strings.HasPrefix(id, "yt_") {
		return a.resumeYouTubeDownload(id)
	}
	if a.manager == nil {
		return fmt.Errorf("download manager is not ready")
	}
	return a.manager.ResumeDownload(a.ctx, id)
}

// CancelDownload cancels a download and optionally removes the downloaded file.
func (a *App) CancelDownload(id string, deleteFile bool) error {
	if strings.HasPrefix(id, "yt_") {
		return a.cancelYouTubeDownload(id, deleteFile)
	}
	if a.manager == nil {
		return fmt.Errorf("download manager is not ready")
	}
	return a.manager.CancelDownload(a.ctx, id, deleteFile)
}

// SetMaxConcurrentDownloads dynamically adjusts the concurrency limit.
func (a *App) SetMaxConcurrentDownloads(limit int) {
	if a.manager != nil {
		a.manager.SetMaxConcurrentDownloads(limit)
	}
}

// GetMaxConcurrentDownloads returns the current concurrency limit.
func (a *App) GetMaxConcurrentDownloads() int {
	if a.manager != nil {
		return a.manager.GetMaxConcurrentDownloads()
	}
	return 3
}

// BatchPauseDownloads suspends multiple download tasks.
func (a *App) BatchPauseDownloads(ids []string) error {
	var lastErr error
	for _, id := range ids {
		if err := a.PauseDownload(id); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// BatchResumeDownloads restarts multiple paused downloads.
func (a *App) BatchResumeDownloads(ids []string) error {
	var lastErr error
	for _, id := range ids {
		if err := a.ResumeDownload(id); err != nil {
			if !errors.Is(err, models.ErrAlreadyRunning) && !errors.Is(err, models.ErrAlreadyCompleted) {
				lastErr = err
			}
		}
	}
	return lastErr
}

// BatchDeleteDownloads cancels multiple downloads and optionally removes downloaded files.
func (a *App) BatchDeleteDownloads(ids []string, deleteFiles bool) error {
	var lastErr error
	for _, id := range ids {
		if err := a.CancelDownload(id, deleteFiles); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// GetDownloads retrieves all downloads from SQLite.
func (a *App) GetDownloads() ([]*models.Download, error) {
	if a.manager == nil {
		return nil, fmt.Errorf("download manager is not ready")
	}
	return a.manager.GetAllDownloads(a.ctx)
}

// GetChunks retrieves chunk status and byte offsets for a specific download.
func (a *App) GetChunks(downloadID string) ([]*models.Chunk, error) {
	if a.repo == nil {
		return nil, fmt.Errorf("storage repository is not ready")
	}
	return a.repo.GetChunks(a.ctx, downloadID)
}

// GetURLHistory retrieves recently downloaded URLs from SQLite.
func (a *App) GetURLHistory() ([]*models.URLHistoryItem, error) {
	if a.manager == nil {
		return nil, fmt.Errorf("download manager is not ready")
	}
	return a.manager.GetURLHistory(a.ctx, 50)
}

// DeleteURLHistoryItem removes a specific record from URL history.
func (a *App) DeleteURLHistoryItem(id int64) error {
	if a.manager == nil {
		return fmt.Errorf("download manager is not ready")
	}
	return a.manager.DeleteURLHistoryItem(a.ctx, id)
}

// ClearURLHistory purges the URL history table.
func (a *App) ClearURLHistory() error {
	if a.manager == nil {
		return fmt.Errorf("download manager is not ready")
	}
	return a.manager.ClearURLHistory(a.ctx)
}

// SelectDirectory opens the native OS directory picker dialog.
func (a *App) SelectDirectory() (string, error) {
	home, _ := os.UserHomeDir()
	defaultDir := filepath.Join(home, "Downloads")
	selected, err := wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		DefaultDirectory: defaultDir,
		Title:            "Select Download Directory",
	})
	if err != nil {
		return "", err
	}
	return selected, nil
}

// GetDefaultDownloadDir returns the system default downloads directory.
func (a *App) GetDefaultDownloadDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	dlPath := filepath.Join(home, "Downloads")
	if _, err := os.Stat(dlPath); err == nil {
		return dlPath
	}
	return home
}

// OpenFile opens the downloaded file with the system default application.
func (a *App) OpenFile(filePath string) error {
	return a.opener.OpenFile(filePath)
}

// ShowInFolder reveals the downloaded file in the native file manager (Finder / Explorer).
func (a *App) ShowInFolder(filePath string) error {
	return a.opener.ShowInFolder(filePath)
}

// OpenFolder opens a directory in the native file manager (Finder / Explorer).
func (a *App) OpenFolder(folderPath string) error {
	if folderPath == "" {
		folderPath = a.GetDefaultDownloadDir()
	}
	_ = os.MkdirAll(folderPath, 0755)
	return a.opener.OpenFile(folderPath)
}

// HandleGrabRequest raises the standalone mini prompt window and emits download:prompt_mode.
func (a *App) HandleGrabRequest(req models.GrabRequest) {
	if a.ctx == nil {
		return
	}

	a.mu.Lock()
	a.isPromptOnly = true
	a.mu.Unlock()

	isYT := ytdl.IsYouTubeURL(req.URL)

	wailsRuntime.WindowSetAlwaysOnTop(a.ctx, true)
	if isYT {
		wailsRuntime.WindowSetMinSize(a.ctx, 520, 480)
		wailsRuntime.WindowSetSize(a.ctx, 560, 520)
	} else {
		wailsRuntime.WindowSetMinSize(a.ctx, 460, 200)
		wailsRuntime.WindowSetSize(a.ctx, 500, 220)
	}
	wailsRuntime.WindowCenter(a.ctx)
	wailsRuntime.EventsEmit(a.ctx, "download:prompt_mode", map[string]any{
		"standalone": true,
		"data":       req,
		"isYouTube":  isYT,
	})
	wailsRuntime.WindowShow(a.ctx)
	wailsRuntime.WindowUnminimise(a.ctx)
}

// ClosePromptWindow hides the standalone prompt window back to tray/dock and dispatches start notification.
func (a *App) ClosePromptWindow(downloadStarted bool, filename string, size int64) {
	if downloadStarted && filename != "" {
		sizeStr := sysutil.FormatBytes(size)
		if a.notifier != nil {
			_ = a.notifier.Notify("Candak Download Started", fmt.Sprintf("Started: %s (%s)", filename, sizeStr))
		}
	}

	a.mu.Lock()
	promptOnly := a.isPromptOnly
	a.isPromptOnly = false
	a.mu.Unlock()

	if promptOnly {
		wailsRuntime.WindowSetAlwaysOnTop(a.ctx, false)
		wailsRuntime.WindowSetMinSize(a.ctx, 800, 500)
		wailsRuntime.WindowSetSize(a.ctx, 1080, 720)
		wailsRuntime.WindowCenter(a.ctx)
		wailsRuntime.EventsEmit(a.ctx, "download:prompt_mode", map[string]any{
			"standalone": false,
		})
		a.SetWindowVisible(false)
		wailsRuntime.Hide(a.ctx)
	}
}

// OpenMainWindow restores the full manager window from system tray/dock.
func (a *App) OpenMainWindow() {
	if a.ctx == nil {
		return
	}

	a.mu.Lock()
	a.isPromptOnly = false
	a.mu.Unlock()

	wailsRuntime.WindowSetAlwaysOnTop(a.ctx, false)
	wailsRuntime.WindowSetMinSize(a.ctx, 800, 500)
	wailsRuntime.WindowSetSize(a.ctx, 1080, 720)
	wailsRuntime.WindowCenter(a.ctx)
	wailsRuntime.EventsEmit(a.ctx, "download:prompt_mode", map[string]any{
		"standalone": false,
	})
	a.SetWindowVisible(true)
	wailsRuntime.Show(a.ctx)
	wailsRuntime.WindowShow(a.ctx)
	wailsRuntime.WindowUnminimise(a.ctx)
}

// NotifyDesktop sends a native desktop notification.
func (a *App) NotifyDesktop(title, message string) {
	if a.notifier != nil {
		_ = a.notifier.Notify(title, message)
	}
}

// ProbeURLWithHeaders analyzes target URL metadata with custom headers (e.g. cookies/referer).
func (a *App) ProbeURLWithHeaders(rawURL string, headers map[string]string) (*models.ProbeResult, error) {
	if a.manager == nil {
		return nil, fmt.Errorf("download manager is not ready")
	}
	return a.manager.ProbeURLWithHeaders(a.ctx, rawURL, headers)
}

// SetClipboardWatching enables or disables the background clipboard watcher.
func (a *App) SetClipboardWatching(enabled bool) bool {
	if a.clipboardWatcher == nil {
		return false
	}
	if enabled {
		a.clipboardWatcher.Start(a.ctx)
	} else {
		a.clipboardWatcher.Stop()
	}
	return a.clipboardWatcher.IsRunning()
}

// IsClipboardWatching returns whether the clipboard watcher is actively polling.
func (a *App) IsClipboardWatching() bool {
	if a.clipboardWatcher == nil {
		return false
	}
	return a.clipboardWatcher.IsRunning()
}

// SuppressClipboardURL instructs the watcher to ignore a specific URL.
func (a *App) SuppressClipboardURL(rawURL string) {
	if a.clipboardWatcher != nil {
		a.clipboardWatcher.SuppressURL(rawURL)
	}
}

// GetLoopbackStatus returns the current loopback server running status and address.
func (a *App) GetLoopbackStatus() map[string]any {
	running := false
	addr := ""
	if a.loopbackServer != nil {
		running = a.loopbackServer.IsRunning()
		addr = a.loopbackServer.Address()
	}
	return map[string]any{
		"running": running,
		"address": addr,
		"port":    server.DefaultPort,
	}
}

// GetExtensionDir returns the absolute path to the browser extension directory.
func (a *App) GetExtensionDir() string {
	// If running in development workspace, prefer local source directory if present
	if cwd, err := os.Getwd(); err == nil && cwd != "/" {
		localPath := filepath.Join(cwd, "browser-extension")
		if stat, err := os.Stat(filepath.Join(localPath, "manifest.json")); err == nil && !stat.IsDir() {
			return localPath
		}
	}

	// In packaged desktop environments (.app / installed), extract embedded extension to user data directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	targetDir := filepath.Join(homeDir, ".candak", "browser-extension")
	_ = a.ensureExtensionFiles(targetDir)
	return targetDir
}

func (a *App) ensureExtensionFiles(targetDir string) error {
	_ = os.MkdirAll(targetDir, 0755)

	return fs.WalkDir(embeddedExtension, "browser-extension", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel("browser-extension", path)
		if err != nil || relPath == "." {
			return nil
		}
		destPath := filepath.Join(targetDir, relPath)
		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}
		data, err := embeddedExtension.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destPath, data, 0644)
	})
}

// SetMinimized flags whether the application was launched into Tray Daemon Mode.
func (a *App) SetMinimized(minimized bool) {
	a.mu.Lock()
	a.isMinimized = minimized
	a.mu.Unlock()

	if a.manager != nil {
		a.manager.SetVisible(!minimized)
	}
}

// SetWindowVisible updates whether the main window is currently visible or minimized.
func (a *App) SetWindowVisible(visible bool) {
	a.mu.Lock()
	a.isMinimized = !visible
	a.mu.Unlock()

	if a.manager != nil {
		a.manager.SetVisible(visible)
	}
}

// GetStartupSettings queries OS autostart configuration for Candak.
func (a *App) GetStartupSettings() (models.StartupConfig, error) {
	if a.autostart == nil {
		return models.StartupConfig{}, fmt.Errorf("autostart manager is not ready")
	}
	return a.autostart.GetStartupSettings()
}

// SetStartupSettings sets or removes OS autostart registration for Candak.
func (a *App) SetStartupSettings(cfg models.StartupConfig) error {
	if a.autostart == nil {
		return fmt.Errorf("autostart manager is not ready")
	}
	return a.autostart.SetStartupSettings(cfg)
}

// GetAppVersion returns the compiled application version.
func (a *App) GetAppVersion() string {
	return updater.Version
}

// CheckForUpdates queries the latest GitHub release and reports whether a newer version exists.
func (a *App) CheckForUpdates() (*updater.CheckResult, error) {
	return updater.Check(a.ctx)
}

// DownloadUpdate enqueues a release installer asset into Candak's own download manager.
func (a *App) DownloadUpdate(assetURL, filename string) (*models.Download, error) {
	if a.manager == nil {
		return nil, fmt.Errorf("download manager is not ready")
	}
	return a.manager.AddDownload(a.ctx, models.AddDownloadRequest{
		URL:      assetURL,
		Filename: filename,
		SaveDir:  a.GetDefaultDownloadDir(),
	})
}

// IsYouTubeURL checks if the provided URL is a recognized YouTube video.
func (a *App) IsYouTubeURL(rawURL string) bool {
	return ytdl.IsYouTubeURL(rawURL)
}

// GetYouTubeMetadata fetches video info, thumbnail, and available resolution formats.
func (a *App) GetYouTubeMetadata(rawURL string) (*ytdl.VideoMetadata, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	return ytdl.FetchMetadata(ctx, rawURL)
}

// DownloadYouTubeVideo starts downloading the selected YouTube video format via yt-dlp.
func (a *App) DownloadYouTubeVideo(rawURL, formatID, saveDir, filename string, totalSize int64) error {
	if saveDir == "" {
		saveDir = a.GetDefaultDownloadDir()
	}
	_ = os.MkdirAll(saveDir, 0755)

	if filename == "" {
		filename = "youtube_download.mp4"
	}
	outPath := filepath.Join(saveDir, filename)

	dlID := fmt.Sprintf("yt_%d", time.Now().UnixNano())
	now := time.Now().UTC().Format(time.RFC3339)
	dl := &models.Download{
		ID:             dlID,
		URL:            rawURL,
		Filename:       filename,
		SavePath:       outPath,
		TotalSize:      totalSize,
		DownloadedSize: 0,
		Status:         models.StatusDownloading,
		TotalChunks:    1,
		Headers:        map[string]string{"format_id": formatID},
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if a.repo != nil {
		_ = a.repo.InsertDownload(context.Background(), dl)
	}
	if a.broadcaster != nil {
		a.broadcaster.Emit("download:status_change", dl)
	}

	task := &ytTask{
		rawURL:    rawURL,
		formatID:  formatID,
		saveDir:   saveDir,
		filename:  filename,
		outPath:   outPath,
		totalSize: totalSize,
	}
	a.startYouTubeTask(dl, task)
	return nil
}

func (a *App) startYouTubeTask(dl *models.Download, t *ytTask) {
	ctx, cancel := context.WithCancel(context.Background())
	a.ytMu.Lock()
	t.cancel = cancel
	a.ytTasks[dl.ID] = t
	a.ytMu.Unlock()

	go func() {
		err := ytdl.DownloadVideo(ctx, t.rawURL, t.formatID, t.outPath, func(pct float64, spd, down, tot, eta int64) {
			dl.DownloadedSize = down
			if tot > 0 && dl.TotalSize <= 0 {
				dl.TotalSize = tot
				if a.repo != nil {
					_ = a.repo.UpdateDownloadTotalSize(context.Background(), dl.ID, tot)
				}
			}
			dl.Speed = spd
			dl.ETA = eta
			dl.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

			if a.repo != nil {
				_ = a.repo.UpdateDownloadProgress(context.Background(), dl.ID, dl.DownloadedSize, dl.Speed, dl.ETA, dl.Status)
			}
			if a.broadcaster != nil {
				a.broadcaster.Emit("download:progress", models.ProgressEvent{
					DownloadID:     dl.ID,
					DownloadedSize: dl.DownloadedSize,
					TotalSize:      dl.TotalSize,
					Progress:       pct,
					Speed:          dl.Speed,
					ETA:            dl.ETA,
					Status:         dl.Status,
				})
			}
		})

		if err != nil {
			if ctx.Err() == context.Canceled {
				return
			}
			dl.Status = models.StatusError
			dl.ErrorMessage = err.Error()
			dl.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			if a.repo != nil {
				_ = a.repo.UpdateDownloadStatus(context.Background(), dl.ID, dl.Status, dl.ErrorMessage)
			}
			if a.broadcaster != nil {
				a.broadcaster.Emit("download:status_change", dl)
			}
			if a.notifier != nil {
				_ = a.notifier.Notify("Candak Download Failed", fmt.Sprintf("Failed: %s", t.filename))
			}
			return
		}

		if fi, statErr := os.Stat(t.outPath); statErr == nil {
			dl.TotalSize = fi.Size()
			dl.DownloadedSize = fi.Size()
		}
		dl.Status = models.StatusCompleted
		dl.Speed = 0
		dl.ETA = 0
		dl.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

		if a.repo != nil {
			_ = a.repo.UpdateDownloadProgress(context.Background(), dl.ID, dl.DownloadedSize, 0, 0, dl.Status)
			_ = a.repo.UpdateDownloadStatus(context.Background(), dl.ID, dl.Status, "")
		}
		if a.broadcaster != nil {
			a.broadcaster.Emit("download:status_change", dl)
			a.broadcaster.Emit("download:progress", models.ProgressEvent{
				DownloadID:     dl.ID,
				DownloadedSize: dl.DownloadedSize,
				TotalSize:      dl.TotalSize,
				Progress:       100.0,
				Speed:          0,
				ETA:            0,
				Status:         dl.Status,
			})
		}
		if a.notifier != nil {
			_ = a.notifier.Notify("Candak Download Complete", fmt.Sprintf("Finished: %s", t.filename))
		}
	}()
}

func (a *App) pauseYouTubeDownload(id string) error {
	a.ytMu.Lock()
	t, exists := a.ytTasks[id]
	if exists && t.cancel != nil {
		t.cancel()
		t.cancel = nil
	}
	a.ytMu.Unlock()

	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	if a.repo != nil {
		_ = a.repo.UpdateDownloadStatus(ctx, id, models.StatusPaused, "")
		dl, _ := a.repo.GetDownload(ctx, id)
		if dl != nil {
			dl.Status = models.StatusPaused
			dl.Speed = 0
			if a.broadcaster != nil {
				a.broadcaster.Emit("download:status_change", dl)
			}
		}
	}
	return nil
}

func (a *App) resumeYouTubeDownload(id string) error {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	if a.repo == nil {
		return fmt.Errorf("storage repository not ready")
	}

	dl, err := a.repo.GetDownload(ctx, id)
	if err != nil {
		return err
	}
	if dl.Status == models.StatusDownloading {
		return models.ErrAlreadyRunning
	}
	if dl.Status == models.StatusCompleted {
		return models.ErrAlreadyCompleted
	}

	a.ytMu.Lock()
	t, exists := a.ytTasks[id]
	if !exists {
		formatID := "bestvideo+bestaudio/best"
		if dl.Headers != nil && dl.Headers["format_id"] != "" {
			formatID = dl.Headers["format_id"]
		}
		t = &ytTask{
			rawURL:    dl.URL,
			formatID:  formatID,
			outPath:   dl.SavePath,
			filename:  dl.Filename,
			totalSize: dl.TotalSize,
		}
		a.ytTasks[id] = t
	}
	a.ytMu.Unlock()

	dl.Status = models.StatusDownloading
	dl.Speed = 0
	dl.ETA = 0
	dl.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	_ = a.repo.UpdateDownloadStatus(ctx, id, models.StatusDownloading, "")
	if a.broadcaster != nil {
		a.broadcaster.Emit("download:status_change", dl)
		prog := float64(0)
		if dl.TotalSize > 0 {
			prog = float64(dl.DownloadedSize) / float64(dl.TotalSize) * 100.0
		}
		a.broadcaster.Emit("download:progress", models.ProgressEvent{
			DownloadID:     dl.ID,
			DownloadedSize: dl.DownloadedSize,
			TotalSize:      dl.TotalSize,
			Progress:       prog,
			Speed:          0,
			ETA:            0,
			Status:         dl.Status,
		})
	}

	a.startYouTubeTask(dl, t)
	return nil
}

func (a *App) cancelYouTubeDownload(id string, deleteFile bool) error {
	a.ytMu.Lock()
	t, exists := a.ytTasks[id]
	if exists && t.cancel != nil {
		t.cancel()
	}
	delete(a.ytTasks, id)
	a.ytMu.Unlock()

	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	if a.repo != nil {
		dl, _ := a.repo.GetDownload(ctx, id)
		if dl != nil && deleteFile {
			_ = os.Remove(dl.SavePath)
			_ = os.Remove(dl.SavePath + ".part")
			_ = os.Remove(dl.SavePath + ".ytdl")
		}
		_ = a.repo.DeleteDownload(ctx, id)
	}
	if a.broadcaster != nil {
		a.broadcaster.Emit("download:deleted", id)
	}
	return nil
}
