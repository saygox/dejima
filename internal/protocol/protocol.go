package protocol

// Message type IDs
const (
	MsgKeyEvent    byte = 0x01
	MsgMouseMove   byte = 0x02
	MsgMouseButton byte = 0x03
	MsgMouseScroll byte = 0x04
	MsgACK         byte = 0x10
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

// ACKEvent represents an acknowledgment from RPi.
type ACKEvent struct {
	Status byte
}

// PingEvent represents a heartbeat ping.
type PingEvent struct{}

// Message wraps a typed event with its message ID.
type Message struct {
	Type    byte
	Payload interface{}
}
