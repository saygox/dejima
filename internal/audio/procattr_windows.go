//go:build windows

package audio

import (
	"context"
	"os/exec"
	"strconv"
	"syscall"
	"time"
)

// hideWindow sets Windows-specific process creation flags to prevent
// a console window from appearing when launching subprocesses.
func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}

// killProcess kills the process and its entire child tree using taskkill.
// On Windows, cmd.Process.Kill() only terminates the direct process, leaving
// child processes (e.g. GStreamer's decodebin children) orphaned.
func killProcess(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	kill := exec.CommandContext(ctx, "taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid))
	kill.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	return kill.Run()
}
