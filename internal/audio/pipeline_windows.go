package audio

import "fmt"

// AudioSourceArgs returns the GStreamer audio source element args for Windows.
// wasapisrc uses device for device selection.
func AudioSourceArgs(deviceID string) []string {
	if deviceID != "" {
		return []string{"wasapisrc", fmt.Sprintf("device=%s", deviceID)}
	}
	return []string{"wasapisrc"}
}
