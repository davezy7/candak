//go:build darwin

package sysutil

import (
	"fmt"
	"os/exec"
)

func notifyOS(title, message string) error {
	script := fmt.Sprintf(`display notification %q with title %q`, message, title)
	cmd := exec.Command("osascript", "-e", script)
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() {
		_ = cmd.Wait()
	}()
	return nil
}
