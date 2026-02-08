//go:build !windows

package video

import "os/exec"

// hideWindow is a no-op on non-Windows platforms.
func hideWindow(_ *exec.Cmd) {}
