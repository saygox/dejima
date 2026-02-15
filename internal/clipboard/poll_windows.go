//go:build windows

package clipboard

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Read returns the current host clipboard content.
func Read() (string, error) {
	out, err := exec.Command("powershell", "-NoProfile", "-Command",
		"[Console]::OutputEncoding = [System.Text.Encoding]::UTF8; Get-Clipboard").Output()
	if err != nil {
		return "", fmt.Errorf("Get-Clipboard: %w", err)
	}
	out = bytes.TrimPrefix(out, []byte{0xEF, 0xBB, 0xBF}) // strip UTF-8 BOM
	return strings.TrimRight(string(out), "\r\n"), nil
}

// Write sets the host clipboard content.
func Write(text string) error {
	cmd := exec.Command("powershell", "-NoProfile", "-Command",
		"[Console]::InputEncoding = [System.Text.Encoding]::UTF8; $input | Set-Clipboard")
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Set-Clipboard: %w", err)
	}
	return nil
}
