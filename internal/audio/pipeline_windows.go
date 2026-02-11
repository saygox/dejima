package audio

import (
	"fmt"
	"strings"
)

// AudioSourceArgs returns the GStreamer audio source element args for Windows.
// wasapi2src uses the modern WASAPI2 API which matches the device IDs
// returned by gst-device-monitor-1.0.
func AudioSourceArgs(deviceID string) []string {
	if deviceID != "" {
		escaped := escapeGstBraces(deviceID)
		return []string{"wasapi2src", fmt.Sprintf("device=%s", escaped)}
	}
	return []string{"wasapi2src"}
}

// escapeGstBraces ensures { and } are backslash-escaped for gst-launch's
// pipeline parser, which treats bare braces as request-pad syntax.
func escapeGstBraces(s string) string {
	// Normalize: strip existing escapes, then re-escape uniformly
	s = strings.ReplaceAll(s, `\{`, `{`)
	s = strings.ReplaceAll(s, `\}`, `}`)
	s = strings.ReplaceAll(s, `{`, `\{`)
	s = strings.ReplaceAll(s, `}`, `\}`)
	return s
}
