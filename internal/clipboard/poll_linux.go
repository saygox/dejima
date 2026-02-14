//go:build linux

package clipboard

import (
	"fmt"
	"os/exec"
	"strings"
)

// Read returns the current host clipboard content.
func Read() (string, error) {
	out, err := exec.Command("xclip", "-selection", "clipboard", "-o").Output()
	if err != nil {
		return "", fmt.Errorf("xclip: %w", err)
	}
	return string(out), nil
}

// Write sets the host clipboard content.
func Write(text string) error {
	cmd := exec.Command("xclip", "-selection", "clipboard")
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("xclip: %w", err)
	}
	return nil
}
