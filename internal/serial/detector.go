package serial

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// FT232 USB VID:PID
const (
	FT232VID = "0403"
	FT232PID = "6001"
)

// DetectFT232 attempts to find the serial port for an FT232 device.
// Returns the port name (e.g., "/dev/tty.usbserial-XXX" on macOS, "COM3" on Windows).
func DetectFT232() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		return detectDarwin()
	case "windows":
		return detectWindows()
	case "linux":
		return detectLinux()
	default:
		return "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

func detectDarwin() (string, error) {
	// Use system_profiler to find USB devices, then match tty
	out, err := exec.Command("sh", "-c",
		`ls /dev/tty.usbserial-* 2>/dev/null || ls /dev/tty.usbmodem* 2>/dev/null`).Output()
	if err != nil {
		return "", fmt.Errorf("no FT232 serial port found")
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			return line, nil
		}
	}
	return "", fmt.Errorf("no FT232 serial port found")
}

func detectWindows() (string, error) {
	// Use WMIC to find serial ports
	cmd := exec.Command("wmic", "path", "Win32_SerialPort", "get", "DeviceID")
	hideWindow(cmd)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to query serial ports: %w", err)
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "COM") {
			return line, nil
		}
	}
	return "", fmt.Errorf("no FT232 serial port found")
}

func detectLinux() (string, error) {
	// Check /dev/serial/by-id/ for FT232
	out, err := exec.Command("sh", "-c",
		fmt.Sprintf(`ls -la /dev/serial/by-id/ 2>/dev/null | grep -i "%s_%s"`, FT232VID, FT232PID)).Output()
	if err != nil {
		// Fallback: look for ttyUSB devices
		out2, err2 := exec.Command("sh", "-c", `ls /dev/ttyUSB* 2>/dev/null`).Output()
		if err2 != nil {
			return "", fmt.Errorf("no FT232 serial port found")
		}
		lines := strings.Split(strings.TrimSpace(string(out2)), "\n")
		if len(lines) > 0 && lines[0] != "" {
			return lines[0], nil
		}
		return "", fmt.Errorf("no FT232 serial port found")
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		parts := strings.Split(line, "../../")
		if len(parts) == 2 {
			return "/dev/" + strings.TrimSpace(parts[1]), nil
		}
	}
	return "", fmt.Errorf("no FT232 serial port found")
}
