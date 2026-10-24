//go:build windows

package sysutil

import (
	"candak/internal/tray"
)

func notifyOS(title, message string) error {
	// Delegate desktop notification to native Windows tray
	_ = tray.ShowNotification(title, message)
	return nil
}
