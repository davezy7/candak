package sysutil

import (
	"fmt"
)

// Notifier defines the interface for native desktop notifications.
type Notifier interface {
	Notify(title, message string) error
}

// OSNotifier sends native desktop notifications via OS utilities.
type OSNotifier struct{}

// NewOSNotifier creates a new OSNotifier.
func NewOSNotifier() *OSNotifier {
	return &OSNotifier{}
}

// Notify triggers a native OS desktop notification without using PowerShell or external scripts.
func (n *OSNotifier) Notify(title, message string) error {
	return notifyOS(title, message)
}

// FormatBytes formats byte counts to human-readable strings.
func FormatBytes(bytes int64) string {
	if bytes <= 0 {
		return "Unknown size"
	}
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
