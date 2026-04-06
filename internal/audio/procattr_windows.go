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
	// Try taskkill /T first to kill process tree
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	kill := exec.CommandContext(ctx, "taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid))
	kill.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	if err := kill.Run(); err != nil {
		// Fallback: kill the process directly
		_ = cmd.Process.Kill()
	}
	return nil
}

// waitDeviceRelease waits for Windows to release WASAPI device handles
// after a GStreamer process has been reaped. WASAPI2 can take over a second
// to fully release exclusive-mode handles.
func waitDeviceRelease() {
	time.Sleep(1500 * time.Millisecond)
}
