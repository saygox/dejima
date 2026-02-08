package protocol

// Message type IDs
const (
	MsgKeyEvent    byte = 0x01
	MsgMouseMove   byte = 0x02
	MsgMouseButton byte = 0x03
	MsgMouseScroll byte = 0x04
	MsgTextInput        byte = 0x05
	MsgClipboardReq     byte = 0x06
	MsgClipboardData    byte = 0x07
	MsgMouseAbs    byte = 0x08
	MsgDiagReq     byte = 0x09
	MsgDiagData    byte = 0x0A
	MsgACK              byte = 0x10
	MsgPing        byte = 0xFF
)

// Key/button states
const (
	StateRelease byte = 0x00
	StatePress   byte = 0x01
)

// Mouse buttons
const (
	ButtonLeft   byte = 0x01
	ButtonRight  byte = 0x02
	ButtonMiddle byte = 0x03
)

// ACK status codes
const (
	ACKOk    byte = 0x00
	ACKError byte = 0x01
)

// KeyEvent represents a keyboard key press/release.
type KeyEvent struct {
	Keycode uint16
	State   byte // StatePress or StateRelease
}

// MouseMoveEvent represents relative mouse movement.
type MouseMoveEvent struct {
	DX int16
	DY int16
}

// MouseButtonEvent represents a mouse button press/release.
type MouseButtonEvent struct {
	Button byte
	State  byte
}

// MouseScrollEvent represents a scroll wheel event.
type MouseScrollEvent struct {
	Delta int8
}

// MouseAbsEvent represents an absolute mouse position (0-32767 normalized).
type MouseAbsEvent struct {
	X uint16
	Y uint16
}

// ACKEvent represents an acknowledgment from RPi.
type ACKEvent struct {
	Status byte
}

// TextInputEvent represents a UTF-8 text string to type on the remote.
type TextInputEvent struct {
	Text  string
	Paste bool // false = type (wtype/xdotool), true = paste (wl-copy + Ctrl+V)
}

// ClipboardDataEvent carries clipboard text from RPi to host.
type ClipboardDataEvent struct {
	Text string
}

// DiagDataEvent carries diagnostic text from RPi to host.
type DiagDataEvent struct {
	Text string
}

// PingEvent represents a heartbeat ping.
type PingEvent struct{}

// Message wraps a typed event with its message ID.
type Message struct {
	Type    byte
	Payload interface{}
}
