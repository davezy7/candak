package tray

// Callbacks holds the user event handlers for system tray actions.
type Callbacks struct {
	OnOpen     func()
	OnPauseAll func()
	OnQuit     func()
}

// Tray defines the system tray lifecycle interface.
type Tray interface {
	Start() error
	Stop()
}
