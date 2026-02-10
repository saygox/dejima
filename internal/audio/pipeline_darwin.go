package audio

import "fmt"

// AudioSourceArgs returns the GStreamer audio source element args for macOS.
// osxaudiosrc uses unique-id for device selection.
func AudioSourceArgs(deviceID string) []string {
	if deviceID != "" {
		return []string{"osxaudiosrc", fmt.Sprintf("unique-id=%s", deviceID)}
	}
	return []string{"osxaudiosrc"}
}
