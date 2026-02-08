package video

import "fmt"

// PipelineSource returns the GStreamer source element for Linux.
func PipelineSource(deviceIndex int) string {
	return fmt.Sprintf("v4l2src device-index=%d", deviceIndex)
}
