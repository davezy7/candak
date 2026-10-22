package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"candak/internal/db"
	"candak/internal/downloader"
	"candak/internal/models"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx             context.Context
	database        *db.Database
	downloadManager *downloader.Manager
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Locate app data directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	appDir := filepath.Join(homeDir, ".candak")
	_ = os.MkdirAll(appDir, 0755)

	dbPath := filepath.Join(appDir, "candak.db")
	database, err := db.InitDB(dbPath)
	if err != nil {
		wailsRuntime.LogErrorf(ctx, "Failed to initialize database: %v", err)
		return
	}
	a.database = database

	a.downloadManager = downloader.NewManager(database)
	a.downloadManager.SetAppContext(ctx)
}

// shutdown is called at application termination
func (a *App) shutdown(ctx context.Context) {
	if a.database != nil {
		_ = a.database.Close()
	}
}

// ProbeURL analyzes a URL for headers, filename, range support, and content size
func (a *App) ProbeURL(rawURL string) (*models.ProbeResult, error) {
	if a.downloadManager == nil {
		return nil, fmt.Errorf("download manager not initialized")
	}
	return a.downloadManager.ProbeURL(rawURL)
}

// AddDownload starts a new download
func (a *App) AddDownload(req models.AddDownloadRequest) (*models.Download, error) {
	if a.downloadManager == nil {
		return nil, fmt.Errorf("download manager not initialized")
	}
	return a.downloadManager.AddDownload(req)
}

// PauseDownload pauses an ongoing download and keeps chunk offsets saved in SQLite
func (a *App) PauseDownload(id string) error {
	if a.downloadManager == nil {
		return fmt.Errorf("download manager not initialized")
	}
	return a.downloadManager.PauseDownload(id)
}

// ResumeDownload resumes a paused or interrupted download from saved chunk offsets
func (a *App) ResumeDownload(id string) error {
	if a.downloadManager == nil {
		return fmt.Errorf("download manager not initialized")
	}
	return a.downloadManager.ResumeDownload(id)
}

// CancelDownload cancels a download and optionally removes the downloaded file
func (a *App) CancelDownload(id string, deleteFile bool) error {
	if a.downloadManager == nil {
		return fmt.Errorf("download manager not initialized")
	}
	return a.downloadManager.CancelDownload(id, deleteFile)
}

// GetDownloads retrieves all downloads from SQLite
func (a *App) GetDownloads() ([]*models.Download, error) {
	if a.downloadManager == nil {
		return nil, fmt.Errorf("download manager not initialized")
	}
	return a.downloadManager.GetAllDownloads()
}

// GetChunks retrieves chunk status and offsets for a specific download
func (a *App) GetChunks(downloadID string) ([]*models.Chunk, error) {
	if a.database == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return a.database.GetChunks(downloadID)
}

// GetURLHistory retrieves recently downloaded URLs from SQLite
func (a *App) GetURLHistory() ([]*models.URLHistoryItem, error) {
	if a.downloadManager == nil {
		return nil, fmt.Errorf("download manager not initialized")
	}
	return a.downloadManager.GetURLHistory(50)
}

// ClearURLHistory clears the URL history table
func (a *App) ClearURLHistory() error {
	if a.downloadManager == nil {
		return fmt.Errorf("download manager not initialized")
	}
	return a.downloadManager.ClearURLHistory()
}

// SelectDirectory opens the native OS directory selector
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

// GetDefaultDownloadDir returns the system default downloads folder
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

// OpenFile opens the downloaded file with system default application
func (a *App) OpenFile(filePath string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", filePath)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", filePath)
	default:
		cmd = exec.Command("xdg-open", filePath)
	}
	return cmd.Start()
}

// ShowInFolder reveals the downloaded file in the OS file manager
func (a *App) ShowInFolder(filePath string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", "-R", filePath)
	case "windows":
		cmd = exec.Command("explorer", "/select,", filePath)
	default:
		cmd = exec.Command("xdg-open", filepath.Dir(filePath))
	}
	return cmd.Start()
}
