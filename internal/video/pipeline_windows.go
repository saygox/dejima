package video

import "fmt"

// PipelineSource returns the GStreamer source element for Windows.
func PipelineSource(deviceIndex int) string {
	return fmt.Sprintf("mfvideosrc device-index=%d", deviceIndex)
}
