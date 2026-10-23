package events

import (
	"context"
	"sync"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// Broadcaster provides an abstraction for emitting asynchronous system events.
type Broadcaster interface {
	// Emit sends an event with payload to listeners.
	Emit(eventName string, data any)
}

// NullBroadcaster is a no-op broadcaster useful for unit testing and headless environments.
type NullBroadcaster struct{}

// NewNullBroadcaster constructs a no-op Broadcaster.
func NewNullBroadcaster() *NullBroadcaster {
	return &NullBroadcaster{}
}

// Emit does nothing in NullBroadcaster.
func (n *NullBroadcaster) Emit(eventName string, data any) {}

// WailsBroadcaster implements Broadcaster by delegating to the Wails runtime.
type WailsBroadcaster struct {
	mu  sync.RWMutex
	ctx context.Context
}

// NewWailsBroadcaster initializes a Broadcaster tied to Wails application context.
func NewWailsBroadcaster() *WailsBroadcaster {
	return &WailsBroadcaster{}
}

// SetContext assigns the Wails application context required for runtime event emission.
func (w *WailsBroadcaster) SetContext(ctx context.Context) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.ctx = ctx
}

// Emit broadcasts the event through Wails runtime if context is active.
func (w *WailsBroadcaster) Emit(eventName string, data any) {
	w.mu.RLock()
	ctx := w.ctx
	w.mu.RUnlock()

	if ctx != nil {
		wailsRuntime.EventsEmit(ctx, eventName, data)
	}
}
