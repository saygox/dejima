package video

import "fmt"

// PipelineSourceArgs returns the GStreamer source element args for Windows.
// Uses ksvideosrc with device-path if available, falls back to device-index.
func PipelineSourceArgs(deviceIndex int, devicePath string) []string {
	if devicePath != "" {
		return []string{"ksvideosrc", fmt.Sprintf("device-path=%s", devicePath)}
	}
	return []string{"ksvideosrc", fmt.Sprintf("device-index=%d", deviceIndex)}
}
