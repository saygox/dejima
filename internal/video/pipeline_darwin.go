package video

import "fmt"

// PipelineSourceArgs returns the GStreamer source element args for macOS.
func PipelineSourceArgs(deviceIndex int) []string {
	return []string{"avfvideosrc", fmt.Sprintf("device-index=%d", deviceIndex)}
}
