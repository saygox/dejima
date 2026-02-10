package audio

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// AudioDevice represents a detected audio capture device.
type AudioDevice struct {
	ID   string `json:"id"`   // platform-specific device identifier (unique-id, device, etc.)
	Name string `json:"name"`
}

// ListDevices runs gst-device-monitor-1.0 to enumerate Audio/Source devices.
func ListDevices() ([]AudioDevice, error) {
	cmd := exec.Command("gst-device-monitor-1.0", "Audio/Source")
	hideWindow(cmd)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("running gst-device-monitor-1.0 Audio/Source: %w", err)
	}
	return parseDeviceMonitorOutput(string(out)), nil
}

// parseDeviceMonitorOutput extracts audio device names and identifiers from
// gst-device-monitor-1.0 output.
// Handles macOS (unique-id=), Linux (device=), Windows (device=), and
// fallback device-index= patterns.
func parseDeviceMonitorOutput(output string) []AudioDevice {
	var devices []AudioDevice
	lines := bytes.Split([]byte(output), []byte("\n"))

	var currentName string
	for _, line := range lines {
		s := string(bytes.TrimSpace(line))

		// "name  : Built-in Microphone"
		if bytes.HasPrefix(bytes.TrimSpace(line), []byte("name")) && bytes.Contains(line, []byte(":")) {
			parts := bytes.SplitN(line, []byte(":"), 2)
			if len(parts) == 2 {
				currentName = string(bytes.TrimSpace(parts[1]))
			}
			continue
		}

		// gst-launch line: extract device identifier
		// "gst-launch-1.0 osxaudiosrc unique-id='...' ! ..."
		// "gst-launch-1.0 pulsesrc device=... ! ..."
		// "gst-launch-1.0 wasapisrc device=... ! ..."
		if currentName != "" && bytes.Contains(line, []byte("gst-launch")) {
			if id := extractDeviceID(s); id != "" {
				devices = append(devices, AudioDevice{
					ID:   id,
					Name: currentName,
				})
				currentName = ""
				continue
			}
		}

		// Reset on next "Device found:"
		if s == "Device found:" {
			currentName = ""
		}
	}

	// Deduplicate by name
	nameIdx := make(map[string]int, len(devices))
	var deduped []AudioDevice
	for _, d := range devices {
		if _, ok := nameIdx[d.Name]; ok {
			continue
		}
		nameIdx[d.Name] = len(deduped)
		deduped = append(deduped, d)
	}

	return deduped
}

// extractDeviceID extracts the device identifier value from a gst-launch line.
// Tries unique-id=, device=, device-path=, device-index= in order.
func extractDeviceID(line string) string {
	// Try each known property key
	for _, key := range []string{"unique-id=", "device-path=", "device="} {
		idx := strings.Index(line, key)
		if idx < 0 {
			continue
		}
		rest := line[idx+len(key):]
		return extractValue(rest)
	}
	// Fallback: device-index=N (convert to string)
	if idx := strings.Index(line, "device-index="); idx >= 0 {
		rest := line[idx+len("device-index="):]
		return extractValue(rest)
	}
	return ""
}

// extractValue extracts a value from a string that may be quoted with single
// or double quotes. Stops at " !" boundary for unquoted values.
func extractValue(s string) string {
	if len(s) == 0 {
		return ""
	}
	if s[0] == '\'' || s[0] == '"' {
		quote := s[0]
		end := strings.IndexByte(s[1:], quote)
		if end >= 0 {
			return s[1 : end+1]
		}
		return s[1:]
	}
	// Unquoted: take until " !" or end of line
	for i, ch := range s {
		if ch == ' ' || ch == '!' {
			return strings.TrimSpace(s[:i])
		}
	}
	return strings.TrimSpace(s)
}
