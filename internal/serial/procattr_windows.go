//go:build windows

package serial

import (
	"os/exec"
	"syscall"
)

// hideWindow sets Windows-specific process creation flags to prevent
// a console window from appearing when launching subprocesses.
func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}
