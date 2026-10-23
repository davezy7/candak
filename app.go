package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"candak/internal/downloader"
	"candak/internal/events"
	"candak/internal/models"
	"candak/internal/planner"
	"candak/internal/prober"
	"candak/internal/storage"
	"candak/internal/storage/sqlite"
	"candak/internal/sysutil"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App acts as the boundary adapter between the Wails frontend IPC and domain services.
type App struct {
	ctx         context.Context
	repo        storage.Repository
	manager     *downloader.Manager
	broadcaster *events.WailsBroadcaster
	opener      sysutil.SystemOpener
}

// NewApp constructs a new App application struct.
func NewApp() *App {
	broadcaster := events.NewWailsBroadcaster()
	return &App{
		broadcaster: broadcaster,
		opener:      sysutil.NewOSOpener(),
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
		wailsRuntime.LogErrorf(ctx, "Failed to initialize SQLite storage: %v", err)
		return
	}
	a.repo = repo

	// Compose Download Manager with injected dependencies (SOLID DIP)
	a.manager = downloader.NewManager(downloader.Config{
		Repository:   repo,
		Prober:       prober.NewHTTPProber(),
		ChunkPlanner: planner.NewDefaultChunkPlanner(),
		Broadcaster:  a.broadcaster,
	})
}

// shutdown is called by Wails when the desktop application terminates.
func (a *App) shutdown(ctx context.Context) {
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
