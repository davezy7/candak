//go:build darwin

package sysutil

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"candak/internal/models"
)

const (
	DarwinLaunchAgentLabel = "com.candak.downloadmanager"
	DarwinLaunchAgentPlist = "com.candak.downloadmanager.plist"
)

// DarwinAutostart registers autostart using macOS user LaunchAgents.
type DarwinAutostart struct {
	plistPath string
	getExe    func() (string, error)
}

func newPlatformAutostart() Autostarter {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	plist := filepath.Join(home, "Library", "LaunchAgents", DarwinLaunchAgentPlist)
	return &DarwinAutostart{
		plistPath: plist,
		getExe:    os.Executable,
	}
}

// NewDarwinAutostartWithConfig allows setting custom plist path and exe resolver for testing.
func NewDarwinAutostartWithConfig(plistPath string, getExe func() (string, error)) *DarwinAutostart {
	if getExe == nil {
		getExe = os.Executable
	}
	return &DarwinAutostart{
		plistPath: plistPath,
		getExe:    getExe,
	}
}

// GetStartupSettings reads the LaunchAgent plist to check if autostart and minimized are configured.
func (d *DarwinAutostart) GetStartupSettings() (models.StartupConfig, error) {
	data, err := os.ReadFile(d.plistPath)
	if err != nil {
		if os.IsNotExist(err) {
			return models.StartupConfig{Enabled: false, Minimized: false}, nil
		}
		return models.StartupConfig{}, err
	}

	content := string(data)
	minimized := strings.Contains(content, "--minimized")
	return models.StartupConfig{
		Enabled:   true,
		Minimized: minimized,
	}, nil
}

// SetStartupSettings writes or removes the user LaunchAgent plist for autostart.
func (d *DarwinAutostart) SetStartupSettings(cfg models.StartupConfig) error {
	if !cfg.Enabled {
		if err := os.Remove(d.plistPath); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}

	exePath, err := d.getExe()
	if err != nil {
		return fmt.Errorf("failed to determine executable path: %w", err)
	}
	absExe, err := filepath.Abs(exePath)
	if err == nil {
		exePath = absExe
	}

	dir := filepath.Dir(d.plistPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create LaunchAgents directory: %w", err)
	}

	var escapedExe bytes.Buffer
	_ = xml.EscapeText(&escapedExe, []byte(exePath))

	var argsBuf bytes.Buffer
	argsBuf.WriteString(fmt.Sprintf("\t\t<string>%s</string>\n", escapedExe.String()))
	if cfg.Minimized {
		argsBuf.WriteString("\t\t<string>--minimized</string>\n")
	}

	plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>%s</string>
	<key>ProgramArguments</key>
	<array>
%s	</array>
	<key>RunAtLoad</key>
	<true/>
</dict>
</plist>
`, DarwinLaunchAgentLabel, argsBuf.String())

	return os.WriteFile(d.plistPath, []byte(plistContent), 0644)
}
