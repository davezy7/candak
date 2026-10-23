package sysutil

import (
	"encoding/base64"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"unicode/utf16"
)

// Notifier defines the interface for native desktop notifications.
type Notifier interface {
	Notify(title, message string) error
}

// OSNotifier sends native desktop notifications via OS command utilities.
type OSNotifier struct{}

// NewOSNotifier creates a new OSNotifier.
func NewOSNotifier() *OSNotifier {
	return &OSNotifier{}
}

func encodeUTF16LEBase64(s string) string {
	u16 := utf16.Encode([]rune(s))
	b := make([]byte, len(u16)*2)
	for i, v := range u16 {
		b[i*2] = byte(v)
		b[i*2+1] = byte(v >> 8)
	}
	return base64.StdEncoding.EncodeToString(b)
}

// Notify triggers a native OS desktop notification.
func (n *OSNotifier) Notify(title, message string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		script := fmt.Sprintf(`display notification %q with title %q`, message, title)
		cmd = exec.Command("osascript", "-e", script)
	case "windows":
		escTitle := strings.ReplaceAll(title, "'", "''")
		escMsg := strings.ReplaceAll(message, "'", "''")
		psScript := fmt.Sprintf(`[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] > $null; $template = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02); $textNodes = $template.GetElementsByTagName('text'); $textNodes.Item(0).AppendChild($template.CreateTextNode('%s')) > $null; $textNodes.Item(1).AppendChild($template.CreateTextNode('%s')) > $null; $toast = [Windows.UI.Notifications.ToastNotification]::new($template); [Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('Candak').Show($toast);`, escTitle, escMsg)
		cmd = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-EncodedCommand", encodeUTF16LEBase64(psScript))
	default:
		cmd = exec.Command("notify-send", title, message)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() {
		_ = cmd.Wait()
	}()
	return nil
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
