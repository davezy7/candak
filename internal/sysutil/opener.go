package sysutil

import (
	"os/exec"
	"path/filepath"
	"runtime"
)

// SystemOpener provides an interface for interacting with native OS file viewers.
type SystemOpener interface {
	// OpenFile opens a file with its registered default application.
	OpenFile(filePath string) error
	// ShowInFolder reveals a file in the system file manager (Finder/Explorer).
	ShowInFolder(filePath string) error
}

// OSOpener implements SystemOpener using native operating system commands.
type OSOpener struct{}

// NewOSOpener creates a new OSOpener.
func NewOSOpener() *OSOpener {
	return &OSOpener{}
}

// OpenFile opens the file with the default desktop viewer.
func (o *OSOpener) OpenFile(filePath string) error {
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

// ShowInFolder reveals the file in Finder (macOS), Explorer (Windows), or default file manager.
func (o *OSOpener) ShowInFolder(filePath string) error {
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
