//go:build !darwin && !windows

package sysutil

import (
	"os/exec"
)

func notifyOS(title, message string) error {
	cmd := exec.Command("notify-send", title, message)
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() {
		_ = cmd.Wait()
	}()
	return nil
}
