package clipboard

import (
	"context"
	"testing"
	"time"
)

func TestClipboardWatcher_Detection(t *testing.T) {
	currentText := ""
	var detected []string

	watcher := NewWatcher(Config{
		Getter: func() (string, error) {
			return currentText, nil
		},
		OnDetect: func(u string) {
			detected = append(detected, u)
		},
		Interval: 10 * time.Millisecond,
	})

	// 1. Regular text - should be ignored
	currentText = "Hello world"
	watcher.Check()
	if len(detected) != 0 {
		t.Fatalf("expected 0 detected, got %d", len(detected))
	}

	// 2. Normal web link without binary extension - should be ignored
	currentText = "https://google.com/search?q=test"
	watcher.Check()
	if len(detected) != 0 {
		t.Fatalf("expected 0 detected, got %d", len(detected))
	}

	// 3. Direct downloadable file link - should be detected
	currentText = "https://releases.ubuntu.com/22.04/ubuntu.iso"
	watcher.Check()
	if len(detected) != 1 || detected[0] != currentText {
		t.Fatalf("expected 1 detected, got %+v", detected)
	}

	// 4. Same text repeated - should not trigger duplicate
	watcher.Check()
	if len(detected) != 1 {
		t.Fatalf("expected still 1 detected, got %d", len(detected))
	}

	// 5. Suppressed URL - should be ignored
	suppressed := "https://example.com/file.zip"
	watcher.SuppressURL(suppressed)
	currentText = suppressed
	watcher.Check()
	if len(detected) != 1 {
		t.Fatalf("suppressed URL was detected, expected 1, got %d", len(detected))
	}

	// 6. Another valid link - should be detected
	currentText = "https://example.com/installer.dmg"
	watcher.Check()
	if len(detected) != 2 || detected[1] != currentText {
		t.Fatalf("expected 2 detected, got %+v", detected)
	}
}

func TestClipboardWatcher_Lifecycle(t *testing.T) {
	watcher := NewWatcher(Config{
		Getter: func() (string, error) {
			return "", nil
		},
		Interval: 50 * time.Millisecond,
	})

	if watcher.IsRunning() {
		t.Fatal("expected not running initially")
	}

	watcher.Start(context.Background())
	if !watcher.IsRunning() {
		t.Fatal("expected running after Start()")
	}

	watcher.Stop()
	if watcher.IsRunning() {
		t.Fatal("expected stopped after Stop()")
	}
}
