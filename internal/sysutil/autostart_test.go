package sysutil

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"candak/internal/models"
)

func TestDarwinAutostart(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("skipping darwin-specific test")
	}

	tempDir, err := os.MkdirTemp("", "candak-autostart-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	plistPath := filepath.Join(tempDir, "LaunchAgents", DarwinLaunchAgentPlist)
	fakeExe := "/Applications/Candak.app/Contents/MacOS/candak"
	autostart := NewDarwinAutostartWithConfig(plistPath, func() (string, error) {
		return fakeExe, nil
	})

	// 1. Initially non-existent
	cfg, err := autostart.GetStartupSettings()
	if err != nil {
		t.Fatalf("unexpected error getting initial settings: %v", err)
	}
	if cfg.Enabled || cfg.Minimized {
		t.Fatalf("expected disabled autostart initially, got %+v", cfg)
	}

	// 2. Enable with Minimized = true
	err = autostart.SetStartupSettings(models.StartupConfig{
		Enabled:   true,
		Minimized: true,
	})
	if err != nil {
		t.Fatalf("failed to set startup settings: %v", err)
	}

	cfg, err = autostart.GetStartupSettings()
	if err != nil {
		t.Fatalf("failed to get updated settings: %v", err)
	}
	if !cfg.Enabled || !cfg.Minimized {
		t.Fatalf("expected enabled and minimized, got %+v", cfg)
	}

	content, err := os.ReadFile(plistPath)
	if err != nil {
		t.Fatalf("failed to read written plist: %v", err)
	}
	if !strings.Contains(string(content), fakeExe) {
		t.Fatalf("plist missing exe path: %s", string(content))
	}
	if !strings.Contains(string(content), "--minimized") {
		t.Fatalf("plist missing --minimized: %s", string(content))
	}

	// 3. Update to Minimized = false
	err = autostart.SetStartupSettings(models.StartupConfig{
		Enabled:   true,
		Minimized: false,
	})
	if err != nil {
		t.Fatalf("failed to update settings to non-minimized: %v", err)
	}

	cfg, err = autostart.GetStartupSettings()
	if err != nil {
		t.Fatalf("failed to get non-minimized settings: %v", err)
	}
	if !cfg.Enabled || cfg.Minimized {
		t.Fatalf("expected enabled and not minimized, got %+v", cfg)
	}

	content, err = os.ReadFile(plistPath)
	if err != nil {
		t.Fatalf("failed to read updated plist: %v", err)
	}
	if strings.Contains(string(content), "--minimized") {
		t.Fatalf("plist should not contain --minimized: %s", string(content))
	}

	// 4. Disable
	err = autostart.SetStartupSettings(models.StartupConfig{
		Enabled:   false,
		Minimized: false,
	})
	if err != nil {
		t.Fatalf("failed to disable autostart: %v", err)
	}

	cfg, err = autostart.GetStartupSettings()
	if err != nil {
		t.Fatalf("failed to get disabled settings: %v", err)
	}
	if cfg.Enabled {
		t.Fatalf("expected autostart to be disabled, got %+v", cfg)
	}
	if _, err := os.Stat(plistPath); !os.IsNotExist(err) {
		t.Fatalf("expected plist file to be removed, got err: %v", err)
	}

	// 5. Verify XML escaping with special characters in executable path
	specialExe := "/Users/test & user/Candak <app>.app/candak"
	autostartSpecial := NewDarwinAutostartWithConfig(plistPath, func() (string, error) {
		return specialExe, nil
	})
	if err := autostartSpecial.SetStartupSettings(models.StartupConfig{Enabled: true}); err != nil {
		t.Fatalf("failed to set startup with special exe: %v", err)
	}
	contentSpecial, err := os.ReadFile(plistPath)
	if err != nil {
		t.Fatalf("failed to read written plist: %v", err)
	}
	if !strings.Contains(string(contentSpecial), "test &amp; user") {
		t.Fatalf("expected XML-escaped path in plist, got: %s", string(contentSpecial))
	}
}
