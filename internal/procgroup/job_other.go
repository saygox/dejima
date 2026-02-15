//go:build !windows

package procgroup

// Init is a no-op on non-Windows platforms.
// On Linux, child processes are cleaned up via Pdeathsig (set in SysProcAttr).
func Init() error { return nil }

// Add is a no-op on non-Windows platforms.
func Add(pid int) error { return nil }
