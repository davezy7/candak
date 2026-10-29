package sysutil

import (
	"candak/internal/models"
)

// Autostarter manages OS autostart configuration across platforms.
type Autostarter interface {
	GetStartupSettings() (models.StartupConfig, error)
	SetStartupSettings(cfg models.StartupConfig) error
}

// NewAutostart constructs the platform-specific autostarter.
func NewAutostart() Autostarter {
	return newPlatformAutostart()
}
