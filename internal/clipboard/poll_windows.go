//go:build windows

package clipboard

import (
	"fmt"
	"os/exec"
	"strings"
)

// Read returns the current host clipboard content.
func Read() (string, error) {
	out, err := exec.Command("powershell", "-NoProfile", "-Command", "Get-Clipboard").Output()
	if err != nil {
		return "", fmt.Errorf("Get-Clipboard: %w", err)
	}
	return strings.TrimRight(string(out), "\r\n"), nil
}

// Write sets the host clipboard content.
func Write(text string) error {
	cmd := exec.Command("powershell", "-NoProfile", "-Command", "Set-Clipboard", "-Value", text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Set-Clipboard: %w", err)
	}
	return nil
}
