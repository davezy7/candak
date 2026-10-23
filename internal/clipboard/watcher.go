package clipboard

import (
	"context"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"
)

// DefaultFileExtensions contains common downloadable binary and media file extensions.
var DefaultFileExtensions = map[string]struct{}{
	".zip": {}, ".rar": {}, ".7z": {}, ".tar": {}, ".gz": {}, ".tgz": {},
	".bz2": {}, ".xz": {}, ".iso": {}, ".dmg": {}, ".pkg": {}, ".exe": {},
	".msi": {}, ".apk": {}, ".bin": {}, ".appimage": {}, ".deb": {}, ".rpm": {},
	".mp4": {}, ".mkv": {}, ".avi": {}, ".mov": {}, ".mp3": {}, ".flac": {},
	".wav": {}, ".pdf": {}, ".epub": {}, ".torrent": {},
}

// TextGetter abstracts retrieval of clipboard text.
type TextGetter func() (string, error)

// URLHandler callback invoked when a valid downloadable URL is detected.
type URLHandler func(detectedURL string)

// Watcher monitors clipboard text changes for downloadable URLs.
type Watcher struct {
	getter        TextGetter
	onDetect      URLHandler
	interval      time.Duration
	extensions    map[string]struct{}
	mu            sync.Mutex
	lastText      string
	suppressedURL string
	cancel        context.CancelFunc
	running       bool
}

// Config provides setup parameters for Watcher.
type Config struct {
	Getter     TextGetter
	OnDetect   URLHandler
	Interval   time.Duration
	Extensions []string
}

// NewWatcher constructs a new clipboard Watcher.
func NewWatcher(cfg Config) *Watcher {
	interval := cfg.Interval
	if interval <= 0 {
		interval = 1 * time.Second
	}

	exts := make(map[string]struct{})
	if len(cfg.Extensions) > 0 {
		for _, e := range cfg.Extensions {
			e = strings.ToLower(strings.TrimSpace(e))
			if !strings.HasPrefix(e, ".") {
				e = "." + e
			}
			exts[e] = struct{}{}
		}
	} else {
		for k, v := range DefaultFileExtensions {
			exts[k] = v
		}
	}

	return &Watcher{
		getter:     cfg.Getter,
		onDetect:   cfg.OnDetect,
		interval:   interval,
		extensions: exts,
	}
}

// SuppressURL tells the watcher to ignore a specific URL (e.g. copied by Candak itself).
func (w *Watcher) SuppressURL(rawURL string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.suppressedURL = strings.TrimSpace(rawURL)
}

// Start begins periodic clipboard checking.
func (w *Watcher) Start(ctx context.Context) {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(ctx)
	w.cancel = cancel
	w.running = true
	w.mu.Unlock()

	go w.loop(ctx)
}

// Stop stops clipboard polling.
func (w *Watcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.running {
		return
	}
	if w.cancel != nil {
		w.cancel()
		w.cancel = nil
	}
	w.running = false
}

// IsRunning reports whether the watcher is active.
func (w *Watcher) IsRunning() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.running
}

func (w *Watcher) loop(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.check()
		}
	}
}

// Check evaluates the current clipboard contents once.
func (w *Watcher) Check() {
	w.check()
}

func (w *Watcher) check() {
	if w.getter == nil {
		return
	}

	text, err := w.getter()
	if err != nil {
		return
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}

	w.mu.Lock()
	if text == w.lastText || text == w.suppressedURL {
		w.mu.Unlock()
		return
	}
	w.lastText = text
	w.suppressedURL = ""
	handler := w.onDetect
	exts := w.extensions
	w.mu.Unlock()

	if isDownloadableURL(text, exts) {
		if handler != nil {
			handler(text)
		}
	}
}

func isDownloadableURL(rawURL string, allowedExts map[string]struct{}) bool {
	u, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}

	ext := strings.ToLower(path.Ext(u.Path))
	if ext == "" {
		return false
	}

	_, ok := allowedExts[ext]
	return ok
}
