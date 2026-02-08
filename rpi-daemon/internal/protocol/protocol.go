package protocol

import "fmt"

// Message type IDs (same as host side)
const (
	MsgKeyEvent    byte = 0x01
	MsgMouseMove   byte = 0x02
	MsgMouseButton byte = 0x03
	MsgMouseScroll byte = 0x04
	MsgTextInput     byte = 0x05
	MsgClipboardReq  byte = 0x06
	MsgClipboardData byte = 0x07
	MsgMouseAbs  byte = 0x08
	MsgDiagReq   byte = 0x09
	MsgDiagData  byte = 0x0A
	MsgACK           byte = 0x10
	MsgPing        byte = 0xFF
)

const (
	StateRelease byte = 0x00
	StatePress   byte = 0x01
)

const (
	ButtonLeft   byte = 0x01
	ButtonRight  byte = 0x02
	ButtonMiddle byte = 0x03
)

const (
	ACKOk    byte = 0x00
	ACKError byte = 0x01
)

type KeyEvent struct {
	Keycode uint16
	State   byte
}

type MouseMoveEvent struct {
	DX int16
	DY int16
}

type MouseButtonEvent struct {
	Button byte
	State  byte
}

type MouseScrollEvent struct {
	Delta int8
}

type MouseAbsEvent struct {
	X uint16
	Y uint16
}

type TextInputEvent struct {
	Text string
}

type Message struct {
	Type    byte
	Payload interface{}
}

// Decode parses a payload byte slice into a Message.
func Decode(payload []byte) (Message, error) {
	if len(payload) == 0 {
		return Message{}, errorf("empty payload")
	}

	msgType := payload[0]
	data := payload[1:]

	switch msgType {
	case MsgKeyEvent:
		if len(data) != 3 {
			return Message{}, errorf("KEY_EVENT expects 3 bytes, got %d", len(data))
		}
		return Message{
			Type: MsgKeyEvent,
			Payload: KeyEvent{
				Keycode: uint16(data[0])<<8 | uint16(data[1]),
				State:   data[2],
			},
		}, nil

	case MsgMouseMove:
		if len(data) != 4 {
			return Message{}, errorf("MOUSE_MOVE expects 4 bytes, got %d", len(data))
		}
		return Message{
			Type: MsgMouseMove,
			Payload: MouseMoveEvent{
				DX: int16(uint16(data[0])<<8 | uint16(data[1])),
				DY: int16(uint16(data[2])<<8 | uint16(data[3])),
			},
		}, nil

	case MsgMouseButton:
		if len(data) != 2 {
			return Message{}, errorf("MOUSE_BUTTON expects 2 bytes, got %d", len(data))
		}
		return Message{
			Type: MsgMouseButton,
			Payload: MouseButtonEvent{
				Button: data[0],
				State:  data[1],
			},
		}, nil

	case MsgMouseScroll:
		if len(data) != 1 {
			return Message{}, errorf("MOUSE_SCROLL expects 1 byte, got %d", len(data))
		}
		return Message{
			Type: MsgMouseScroll,
			Payload: MouseScrollEvent{
				Delta: int8(data[0]),
			},
		}, nil

	case MsgMouseAbs:
		if len(data) != 4 {
			return Message{}, errorf("MOUSE_ABS expects 4 bytes, got %d", len(data))
		}
		return Message{
			Type: MsgMouseAbs,
			Payload: MouseAbsEvent{
				X: uint16(data[0])<<8 | uint16(data[1]),
				Y: uint16(data[2])<<8 | uint16(data[3]),
			},
		}, nil

	case MsgClipboardReq:
		return Message{Type: MsgClipboardReq}, nil

	case MsgTextInput:
		return Message{
			Type:    MsgTextInput,
			Payload: TextInputEvent{Text: string(data)},
		}, nil

	case MsgDiagReq:
		return Message{Type: MsgDiagReq}, nil

	case MsgPing:
		return Message{Type: MsgPing}, nil

	default:
		return Message{}, errorf("unknown message type: 0x%02x", msgType)
	}
}

// EncodeACK creates an ACK response payload.
func EncodeACK(status byte) []byte {
	return []byte{MsgACK, status}
}

// EncodePing creates a PING payload.
func EncodePing() []byte {
	return []byte{MsgPing}
}

// EncodeDiagData creates a DIAG_DATA response payload.
func EncodeDiagData(text string) []byte {
	data := []byte(text)
	buf := make([]byte, 1+len(data))
	buf[0] = MsgDiagData
	copy(buf[1:], data)
	return buf
}

// EncodeClipboardData creates a CLIPBOARD_DATA response payload.
func EncodeClipboardData(text string) []byte {
	data := []byte(text)
	buf := make([]byte, 1+len(data))
	buf[0] = MsgClipboardData
	copy(buf[1:], data)
	return buf
}

func errorf(format string, args ...interface{}) error {
	return fmt.Errorf(format, args...)
}
