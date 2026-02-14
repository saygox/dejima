package hid

import (
	"fmt"
	"log"
	"sync"

	"github.com/saygox/dejima/internal/protocol"
	"github.com/saygox/dejima/internal/serial"
)

// Controller routes frontend input events through the protocol encoder
// to the serial port.
type Controller struct {
	port *serial.Port
	mu   sync.RWMutex
}

// NewController creates a new HID controller.
func NewController() *Controller {
	return &Controller{}
}

// SetPort sets the serial port for sending HID events.
func (c *Controller) SetPort(port *serial.Port) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.port = port
}

// IsConnected returns whether a serial port is configured.
func (c *Controller) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.port != nil
}

// SendKeyEvent sends a keyboard event.
func (c *Controller) SendKeyEvent(jsCode string, pressed bool) error {
	keycode, ok := LookupKeycode(jsCode)
	if !ok {
		log.Printf("hid: unmapped JS key code: %s (ignored)", jsCode)
		return nil // Don't return error for unmapped keys; just ignore
	}

	state := protocol.StateRelease
	if pressed {
		state = protocol.StatePress
	}

	msg := protocol.Message{
		Type: protocol.MsgKeyEvent,
		Payload: protocol.KeyEvent{
			Keycode: keycode,
			State:   state,
		},
	}

	return c.send(msg)
}

// SendMouseMove sends a relative mouse movement event.
func (c *Controller) SendMouseMove(dx, dy int) error {
	msg := protocol.Message{
		Type: protocol.MsgMouseMove,
		Payload: protocol.MouseMoveEvent{
			DX: int16(dx),
			DY: int16(dy),
		},
	}

	return c.send(msg)
}

// SendMouseButton sends a mouse button press/release event.
func (c *Controller) SendMouseButton(jsButton int, pressed bool) error {
	state := protocol.StateRelease
	if pressed {
		state = protocol.StatePress
	}

	msg := protocol.Message{
		Type: protocol.MsgMouseButton,
		Payload: protocol.MouseButtonEvent{
			Button: JSButtonToProtocol(jsButton),
			State:  state,
		},
	}

	return c.send(msg)
}

// SendMouseAbs sends an absolute mouse position (0-32767 normalized).
func (c *Controller) SendMouseAbs(x, y int) error {
	msg := protocol.Message{
		Type: protocol.MsgMouseAbs,
		Payload: protocol.MouseAbsEvent{
			X: uint16(x),
			Y: uint16(y),
		},
	}
	return c.send(msg)
}

// SendMouseScroll sends a scroll wheel event.
func (c *Controller) SendMouseScroll(delta int) error {
	msg := protocol.Message{
		Type: protocol.MsgMouseScroll,
		Payload: protocol.MouseScrollEvent{
			Delta: int8(delta),
		},
	}

	return c.send(msg)
}

// SendText sends a text string to the remote.
// paste=false uses wtype/xdotool (type mode), paste=true uses wl-copy+Ctrl+V (paste mode).
// Long text is split into chunks to fit within the serial frame limit.
func (c *Controller) SendText(text string, paste bool) error {
	if text == "" {
		return nil
	}

	// Max payload = 256 bytes, minus 1 byte msg type, minus 1 byte mode = 254 bytes for text.
	const maxChunk = 199 // conservative to stay well within frame limit
	data := []byte(text)

	for len(data) > 0 {
		end := maxChunk
		if end > len(data) {
			end = len(data)
		}
		// Don't split in the middle of a UTF-8 character
		for end > 0 && end < len(data) && (data[end]&0xC0) == 0x80 {
			end--
		}
		if end == 0 {
			end = maxChunk // shouldn't happen, but avoid infinite loop
		}
		chunk := string(data[:end])
		data = data[end:]

		final := len(data) == 0
		log.Printf("hid: sending text chunk paste=%v final=%v len=%d", paste, final, len(chunk))
		msg := protocol.Message{
			Type: protocol.MsgTextInput,
			Payload: protocol.TextInputEvent{
				Text:  chunk,
				Paste: paste,
				Final: final,
			},
		}
		if err := c.send(msg); err != nil {
			return err
		}
	}
	return nil
}

// SendPing sends a heartbeat ping.
func (c *Controller) SendPing() error {
	msg := protocol.Message{
		Type:    protocol.MsgPing,
		Payload: protocol.PingEvent{},
	}
	return c.send(msg)
}

func (c *Controller) send(msg protocol.Message) error {
	c.mu.RLock()
	port := c.port
	c.mu.RUnlock()

	if port == nil {
		return fmt.Errorf("serial port not connected")
	}

	payload, err := protocol.Encode(msg)
	if err != nil {
		return fmt.Errorf("encoding message: %w", err)
	}

	if err := port.Write(payload); err != nil {
		log.Printf("hid: send error: %v", err)
		return fmt.Errorf("sending message: %w", err)
	}

	return nil
}
