//go:build !windows

package sysenc

// Decode returns the input unchanged on non-Windows platforms,
// where subprocess output is already UTF-8.
func Decode(b []byte) string {
	return string(b)
}
