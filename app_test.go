package main

import (
	"os"
	"path/filepath"
	"testing"

	"candak/internal/models"
)

func TestEnsureExtensionFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "candak-ext-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	app := NewApp()
	targetDir := filepath.Join(tempDir, "browser-extension")
	if err := app.ensureExtensionFiles(targetDir); err != nil {
		t.Fatalf("ensureExtensionFiles failed: %v", err)
	}

	manifestPath := filepath.Join(targetDir, "manifest.json")
	if _, err := os.Stat(manifestPath); err != nil {
		t.Errorf("manifest.json not extracted: %v", err)
	}

	dir := app.GetExtensionDir()
	if _, err := os.Stat(filepath.Join(dir, "manifest.json")); err != nil {
		t.Errorf("GetExtensionDir does not contain manifest.json: %v", err)
	}
}

type mockAutostarter struct {
	cfg models.StartupConfig
}

func (m *mockAutostarter) GetStartupSettings() (models.StartupConfig, error) {
	return m.cfg, nil
}

func (m *mockAutostarter) SetStartupSettings(cfg models.StartupConfig) error {
	m.cfg = cfg
	return nil
}

func TestAppStartupSettings(t *testing.T) {
	app := NewApp()
	mock := &mockAutostarter{
		cfg: models.StartupConfig{Enabled: false, Minimized: false},
	}
	app.autostart = mock

	// Test SetMinimized
	app.SetMinimized(true)
	if !app.isMinimized {
		t.Fatalf("expected app.isMinimized to be true")
	}

	// Test SetWindowVisible
	app.SetWindowVisible(true)
	if app.isMinimized {
		t.Fatalf("expected app.isMinimized to be false after SetWindowVisible(true)")
	}
	app.SetWindowVisible(false)
	if !app.isMinimized {
		t.Fatalf("expected app.isMinimized to be true after SetWindowVisible(false)")
	}

	// Test GetStartupSettings
	cfg, err := app.GetStartupSettings()
	if err != nil {
		t.Fatalf("GetStartupSettings failed: %v", err)
	}
	if cfg.Enabled {
		t.Fatalf("expected Enabled=false initially")
	}

	// Test SetStartupSettings
	err = app.SetStartupSettings(models.StartupConfig{Enabled: true, Minimized: true})
	if err != nil {
		t.Fatalf("SetStartupSettings failed: %v", err)
	}

	cfg, err = app.GetStartupSettings()
	if err != nil {
		t.Fatalf("GetStartupSettings failed: %v", err)
	}
	if !cfg.Enabled || !cfg.Minimized {
		t.Fatalf("expected Enabled=true, Minimized=true, got %+v", cfg)
	}
}

func TestAppBatchMethods(t *testing.T) {
	app := NewApp()
	// Manager not initialized yet
	_, err := app.BatchProbeURLs([]string{"http://example.com"})
	if err == nil {
		t.Errorf("expected error when manager not initialized")
	}

	_, err = app.AddBatchDownloads([]models.DownloadRequest{{URL: "http://example.com"}})
	if err == nil {
		t.Errorf("expected error when manager not initialized")
	}

	if err := app.BatchPauseDownloads([]string{"id1"}); err == nil {
		t.Errorf("expected error when manager not initialized")
	}
	if err := app.BatchResumeDownloads([]string{"id1"}); err == nil {
		t.Errorf("expected error when manager not initialized")
	}
	if err := app.BatchDeleteDownloads([]string{"id1"}, false); err == nil {
		t.Errorf("expected error when manager not initialized")
	}
}
