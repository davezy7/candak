//go:build !darwin && !windows

package sysutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"candak/internal/models"
)

// OtherAutostart handles autostart for Linux and other UNIX systems via XDG.
type OtherAutostart struct {
	desktopPath string
	getExe      func() (string, error)
}

func newPlatformAutostart() Autostarter {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	dest := filepath.Join(home, ".config", "autostart", "candak.desktop")
	return &OtherAutostart{
		desktopPath: dest,
		getExe:      os.Executable,
	}
}

// GetStartupSettings reads the XDG autostart desktop entry.
func (o *OtherAutostart) GetStartupSettings() (models.StartupConfig, error) {
	data, err := os.ReadFile(o.desktopPath)
	if err != nil {
		if os.IsNotExist(err) {
			return models.StartupConfig{Enabled: false, Minimized: false}, nil
		}
		return models.StartupConfig{}, err
	}
	content := string(data)
	return models.StartupConfig{
		Enabled:   true,
		Minimized: strings.Contains(content, "--minimized"),
	}, nil
}

// SetStartupSettings writes or removes the XDG autostart desktop entry.
func (o *OtherAutostart) SetStartupSettings(cfg models.StartupConfig) error {
	if !cfg.Enabled {
		if err := os.Remove(o.desktopPath); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}

	exePath, err := o.getExe()
	if err != nil {
		return fmt.Errorf("failed to determine executable path: %w", err)
	}

	dir := filepath.Dir(o.desktopPath)
	_ = os.MkdirAll(dir, 0755)

	execCmd := fmt.Sprintf("\"%s\"", exePath)
	if cfg.Minimized {
		execCmd += " --minimized"
	}

	content := fmt.Sprintf("[Desktop Entry]\nType=Application\nName=Candak\nExec=%s\nHidden=false\nNoDisplay=false\nX-GNOME-Autostart-enabled=true\n", execCmd)
	return os.WriteFile(o.desktopPath, []byte(content), 0644)
}
