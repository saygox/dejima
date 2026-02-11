package lock

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// lockPath returns the path to the lock file.
func lockPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("getting config dir: %w", err)
	}
	return filepath.Join(dir, "dejima", "app.lock"), nil
}

// Acquire obtains a single-instance lock. If another instance is already
// running, it returns an error. On success it returns a release function
// that removes the lock file.
func Acquire() (release func(), err error) {
	path, err := lockPath()
	if err != nil {
		return nil, err
	}

	// Ensure the directory exists.
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("creating lock dir: %w", err)
	}

	// Check for an existing lock file.
	data, err := os.ReadFile(path)
	if err == nil {
		pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
		if err == nil && pid > 0 && processExists(pid) {
			return nil, fmt.Errorf("another instance is already running (pid %d)", pid)
		}
		// Stale lock — process is dead. Fall through to overwrite.
	}

	// Write our PID.
	if err := os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())), 0644); err != nil {
		return nil, fmt.Errorf("writing lock file: %w", err)
	}

	return func() { os.Remove(path) }, nil
}
