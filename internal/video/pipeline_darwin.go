package video

import "fmt"

// PipelineSource returns the GStreamer source element for macOS.
func PipelineSource(deviceIndex int) string {
	return fmt.Sprintf("avfvideosrc device-index=%d", deviceIndex)
}
