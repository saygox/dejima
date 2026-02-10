package audio

import "fmt"

// AudioSourceArgs returns the GStreamer audio source element args for Linux.
// pulsesrc uses device for device selection.
func AudioSourceArgs(deviceID string) []string {
	if deviceID != "" {
		return []string{"pulsesrc", fmt.Sprintf("device=%s", deviceID)}
	}
	return []string{"pulsesrc"}
}
