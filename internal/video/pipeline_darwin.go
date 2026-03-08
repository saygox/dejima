package video

import "fmt"

// PipelineSourceArgs returns the GStreamer source element args for macOS.
func PipelineSourceArgs(deviceIndex int, _ string) []string {
	return []string{"avfvideosrc", fmt.Sprintf("device-index=%d", deviceIndex)}
}

// DiagSourceElement returns the platform-specific video source element name for diagnostics.
func DiagSourceElement() string {
	return "avfvideosrc"
}

// PipelineNeedsDecodebin returns false on macOS because avfvideosrc outputs raw video.
func PipelineNeedsDecodebin() bool {
	return false
}

// PipelineSupportsPassthrough returns true — most USB capture devices output
// MJPEG which can be passed through without decode/encode.
func PipelineSupportsPassthrough() bool {
	return true
}
