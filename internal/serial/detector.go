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

// DetectSerialPort attempts to find a serial port for Bluetooth SPP or FT232 devices.
// Returns the port name (e.g., "/dev/tty.usbserial-XXX" on macOS, "COM3" on Windows).
func DetectSerialPort() (string, error) {
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
		return "", fmt.Errorf("no serial port found")
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			return line, nil
		}
	}
	return "", fmt.Errorf("no serial port found")
}

func detectWindows() (string, error) {
	// Use PowerShell Get-CimInstance to find all COM ports including Bluetooth SPP.
	// Output format: "Name|DeviceID" per line so we can distinguish outgoing
	// Bluetooth ports (have VID& in DeviceID) from incoming ones (LOCALMFG).
	cmd := exec.Command("powershell", "-NoProfile", "-Command",
		`Get-CimInstance Win32_PnPEntity -Filter "Name LIKE '%(COM%'" | ForEach-Object { $_.Name + '|' + $_.DeviceID }`)
	hideWindow(cmd)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to query serial ports: %w", err)
	}

	var btPort, ftPort, anyPort string
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Split "Name|DeviceID"
		parts := strings.SplitN(line, "|", 2)
		name := parts[0]
		devID := ""
		if len(parts) == 2 {
			devID = parts[1]
		}

		port := extractCOMPort(name)
		if port == "" {
			continue
		}
		lowerName := strings.ToLower(name)
		upperDevID := strings.ToUpper(devID)
		switch {
		case strings.Contains(lowerName, "bluetooth"):
			// Skip incoming (local) BT ports — only pick outgoing ones.
			// Incoming ports have LOCALMFG in the DeviceID.
			if strings.Contains(upperDevID, "LOCALMFG") {
				continue
			}
			if btPort == "" {
				btPort = port
			}
		case strings.Contains(lowerName, "ft232") || strings.Contains(lowerName, "ftdi"):
			if ftPort == "" {
				ftPort = port
			}
		default:
			if anyPort == "" {
				anyPort = port
			}
		}
	}

	// Priority: Bluetooth SPP first, then FT232/FTDI, then any COM port
	if btPort != "" {
		return btPort, nil
	}
	if ftPort != "" {
		return ftPort, nil
	}
	if anyPort != "" {
		return anyPort, nil
	}
	return "", fmt.Errorf("no serial port found (checked Bluetooth SPP and USB)")
}

// extractCOMPort extracts "COMn" from a string like "Standard Serial over Bluetooth link (COM7)".
func extractCOMPort(s string) string {
	// Search for "(COM" directly in the original string to avoid index mismatch
	// from multibyte characters when using ToUpper.
	idx := strings.Index(s, "(COM")
	if idx < 0 {
		return ""
	}
	rest := s[idx+1:]
	end := strings.Index(rest, ")")
	if end < 0 {
		return ""
	}
	return rest[:end]
}

func detectLinux() (string, error) {
	// Check /dev/serial/by-id/ for FT232
	out, err := exec.Command("sh", "-c",
		fmt.Sprintf(`ls -la /dev/serial/by-id/ 2>/dev/null | grep -i "%s_%s"`, FT232VID, FT232PID)).Output()
	if err != nil {
		// Fallback: look for ttyUSB devices
		out2, err2 := exec.Command("sh", "-c", `ls /dev/ttyUSB* 2>/dev/null`).Output()
		if err2 != nil {
			return "", fmt.Errorf("no serial port found")
		}
		lines := strings.Split(strings.TrimSpace(string(out2)), "\n")
		if len(lines) > 0 && lines[0] != "" {
			return lines[0], nil
		}
		return "", fmt.Errorf("no serial port found")
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		parts := strings.Split(line, "../../")
		if len(parts) == 2 {
			return "/dev/" + strings.TrimSpace(parts[1]), nil
		}
	}
	return "", fmt.Errorf("no serial port found")
}
