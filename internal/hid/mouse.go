package hid

import "github.com/saygox/dejima/internal/protocol"

// JSButtonToProtocol maps JavaScript MouseEvent.button to protocol button codes.
func JSButtonToProtocol(jsButton int) byte {
	switch jsButton {
	case 0:
		return protocol.ButtonLeft
	case 1:
		return protocol.ButtonMiddle
	case 2:
		return protocol.ButtonRight
	default:
		return protocol.ButtonLeft
	}
}
