//go:build windows

package sysutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"candak/internal/models"
	"golang.org/x/sys/windows/registry"
)

const (
	WindowsRunKeyName      = "Candak"
	WindowsRunRegistryPath = `Software\Microsoft\Windows\CurrentVersion\Run`
)

// WindowsAutostart registers autostart using HKCU\Software\Microsoft\Windows\CurrentVersion\Run.
type WindowsAutostart struct {
	keyPath string
	getExe  func() (string, error)
}

func newPlatformAutostart() Autostarter {
	return &WindowsAutostart{
		keyPath: WindowsRunRegistryPath,
		getExe:  os.Executable,
	}
}

// NewWindowsAutostartWithConfig allows setting custom registry path and exe resolver for testing.
func NewWindowsAutostartWithConfig(keyPath string, getExe func() (string, error)) *WindowsAutostart {
	if getExe == nil {
		getExe = os.Executable
	}
	return &WindowsAutostart{
		keyPath: keyPath,
		getExe:  getExe,
	}
}

// GetStartupSettings reads the Windows Run registry key to check if autostart and minimized are configured.
func (w *WindowsAutostart) GetStartupSettings() (models.StartupConfig, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, w.keyPath, registry.QUERY_VALUE)
	if err != nil {
		if err == registry.ErrNotExist {
			return models.StartupConfig{Enabled: false, Minimized: false}, nil
		}
		return models.StartupConfig{}, err
	}
	defer k.Close()

	val, _, err := k.GetStringValue(WindowsRunKeyName)
	if err != nil {
		if err == registry.ErrNotExist {
			return models.StartupConfig{Enabled: false, Minimized: false}, nil
		}
		return models.StartupConfig{}, err
	}

	minimized := strings.Contains(val, "--minimized")
	return models.StartupConfig{
		Enabled:   true,
		Minimized: minimized,
	}, nil
}

// SetStartupSettings sets or deletes the Candak Run key in the Windows Registry.
func (w *WindowsAutostart) SetStartupSettings(cfg models.StartupConfig) error {
	if !cfg.Enabled {
		k, err := registry.OpenKey(registry.CURRENT_USER, w.keyPath, registry.SET_VALUE)
		if err != nil {
			if err == registry.ErrNotExist {
				return nil
			}
			return err
		}
		defer k.Close()

		if err := k.DeleteValue(WindowsRunKeyName); err != nil && err != registry.ErrNotExist {
			return err
		}
		return nil
	}

	k, _, err := registry.CreateKey(registry.CURRENT_USER, w.keyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	exePath, err := w.getExe()
	if err != nil {
		return fmt.Errorf("failed to determine executable path: %w", err)
	}
	absExe, err := filepath.Abs(exePath)
	if err == nil {
		exePath = absExe
	}

	var cmd string
	if cfg.Minimized {
		cmd = fmt.Sprintf("\"%s\" --minimized", exePath)
	} else {
		cmd = fmt.Sprintf("\"%s\"", exePath)
	}

	return k.SetStringValue(WindowsRunKeyName, cmd)
}
