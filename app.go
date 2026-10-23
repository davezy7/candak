package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

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
	"sync"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App acts as the boundary adapter between the Wails frontend IPC and domain services.
type App struct {
	ctx              context.Context
	repo             storage.Repository
	manager          *downloader.Manager
	broadcaster      *events.WailsBroadcaster
	opener           sysutil.SystemOpener
	notifier         sysutil.Notifier
	loopbackServer   *server.LoopbackServer
	clipboardWatcher *clipboard.Watcher
	mu               sync.Mutex
	isPromptOnly     bool
}

// NewApp constructs a new App application struct.
func NewApp() *App {
	broadcaster := events.NewWailsBroadcaster()
	return &App{
		broadcaster: broadcaster,
		opener:      sysutil.NewOSOpener(),
		notifier:    sysutil.NewOSNotifier(),
	}
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
}

// shutdown is called by Wails when the desktop application terminates.
func (a *App) shutdown(ctx context.Context) {
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

// AddDownload initializes, stores, and starts a download task.
func (a *App) AddDownload(req models.AddDownloadRequest) (*models.Download, error) {
	if a.manager == nil {
		return nil, fmt.Errorf("download manager is not ready")
	}
	return a.manager.AddDownload(a.ctx, req)
}

// PauseDownload suspends an active download task and persists chunk offsets.
func (a *App) PauseDownload(id string) error {
	if a.manager == nil {
		return fmt.Errorf("download manager is not ready")
	}
	return a.manager.PauseDownload(a.ctx, id)
}

// ResumeDownload restarts a paused download using saved chunk offsets.
func (a *App) ResumeDownload(id string) error {
	if a.manager == nil {
		return fmt.Errorf("download manager is not ready")
	}
	return a.manager.ResumeDownload(a.ctx, id)
}

// CancelDownload cancels a download and optionally removes the downloaded file.
func (a *App) CancelDownload(id string, deleteFile bool) error {
	if a.manager == nil {
		return fmt.Errorf("download manager is not ready")
	}
	return a.manager.CancelDownload(a.ctx, id, deleteFile)
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

	wailsRuntime.WindowSetAlwaysOnTop(a.ctx, true)
	wailsRuntime.WindowSetMinSize(a.ctx, 460, 200)
	wailsRuntime.WindowSetSize(a.ctx, 500, 220)
	wailsRuntime.WindowCenter(a.ctx)
	wailsRuntime.EventsEmit(a.ctx, "download:prompt_mode", map[string]any{
		"standalone": true,
		"data":       req,
	})
	wailsRuntime.WindowShow(a.ctx)
	wailsRuntime.WindowUnminimise(a.ctx)
}

// ClosePromptWindow hides the standalone prompt window back to tray and dispatches start notification.
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
		wailsRuntime.WindowHide(a.ctx)
		wailsRuntime.WindowSetMinSize(a.ctx, 800, 500)
		wailsRuntime.WindowSetSize(a.ctx, 1080, 720)
	}
}

// OpenMainWindow restores the full manager window from system tray/dock.
func (a *App) OpenMainWindow() {
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

// GetExtensionDir returns the absolute path to the bundled browser extension directory.
func (a *App) GetExtensionDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "browser-extension"
	}
	return filepath.Join(cwd, "browser-extension")
}
