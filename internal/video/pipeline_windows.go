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

// DiagSourceElement returns the platform-specific video source element name for diagnostics.
func DiagSourceElement() string {
	return "ksvideosrc"
}

// PipelineNeedsDecodebin returns true on Windows because USB capture devices
// often output MJPEG (image/jpeg) which videoconvert cannot handle directly.
// decodebin auto-detects the format and decodes to raw video.
func PipelineNeedsDecodebin() bool {
	return true
}
