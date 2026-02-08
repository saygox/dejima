package video

import "fmt"

// PipelineSourceArgs returns the GStreamer source element args for Windows.
func PipelineSourceArgs(deviceIndex int) []string {
	return []string{"mfvideosrc", fmt.Sprintf("device-index=%d", deviceIndex)}
}
