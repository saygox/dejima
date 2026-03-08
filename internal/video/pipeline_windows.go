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

// PipelineNeedsDecodebin returns false on Windows; the passthrough path
// handles MJPEG natively and the fallback path uses decodebin explicitly.
func PipelineNeedsDecodebin() bool {
	return false
}

// PipelineSupportsPassthrough returns true — USB capture devices on Windows
// typically output MJPEG which can be passed through without decode/encode.
func PipelineSupportsPassthrough() bool {
	return true
}
