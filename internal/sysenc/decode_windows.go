//go:build windows

package sysenc

import (
	"golang.org/x/sys/windows"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/transform"

	"io"
	"strings"
)

// Decode converts subprocess output from the Windows system codepage (ACP)
// to UTF-8. Falls back to raw bytes if the codepage is not recognized.
func Decode(b []byte) string {
	acp := windows.GetACP()
	var reader io.Reader
	switch acp {
	case 932:
		reader = transform.NewReader(strings.NewReader(string(b)), japanese.ShiftJIS.NewDecoder())
	case 949:
		reader = transform.NewReader(strings.NewReader(string(b)), korean.EUCKR.NewDecoder())
	case 936:
		reader = transform.NewReader(strings.NewReader(string(b)), simplifiedchinese.GBK.NewDecoder())
	case 950:
		reader = transform.NewReader(strings.NewReader(string(b)), traditionalchinese.Big5.NewDecoder())
	case 65001:
		return string(b) // already UTF-8
	default:
		return string(b) // best effort
	}
	out, err := io.ReadAll(reader)
	if err != nil {
		return string(b)
	}
	return string(out)
}
