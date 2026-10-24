//go:build !windows

package tray

type dummyTray struct{}

// NewTray constructs a no-op Tray on non-Windows platforms.
func NewTray(callbacks Callbacks) Tray {
	return &dummyTray{}
}

func (d *dummyTray) Start() error {
	return nil
}

func (d *dummyTray) Stop() {}
