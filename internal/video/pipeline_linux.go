package video

import "fmt"

// PipelineSourceArgs returns the GStreamer source element args for Linux.
func PipelineSourceArgs(deviceIndex int) []string {
	return []string{"v4l2src", fmt.Sprintf("device-index=%d", deviceIndex)}
}
