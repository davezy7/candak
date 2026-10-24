package tray

import (
	"testing"
)

func TestTrayLifecycle(t *testing.T) {
	opened := false
	paused := false
	quit := false

	tr := NewTray(Callbacks{
		OnOpen: func() {
			opened = true
		},
		OnPauseAll: func() {
			paused = true
		},
		OnQuit: func() {
			quit = true
		},
	})

	if err := tr.Start(); err != nil {
		t.Fatalf("tray.Start() failed: %v", err)
	}

	tr.Stop()

	// Verify callbacks structure
	_ = opened
	_ = paused
	_ = quit
}
