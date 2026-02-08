package protocol

import "fmt"

// Decode parses a payload byte slice into a Message.
// The first byte of payload must be the message type ID.
func Decode(payload []byte) (Message, error) {
	if len(payload) == 0 {
		return Message{}, fmt.Errorf("empty payload")
	}

	msgType := payload[0]
	data := payload[1:]

	switch msgType {
	case MsgKeyEvent:
		if len(data) != 3 {
			return Message{}, fmt.Errorf("KEY_EVENT expects 3 bytes, got %d", len(data))
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
			return Message{}, fmt.Errorf("MOUSE_MOVE expects 4 bytes, got %d", len(data))
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
			return Message{}, fmt.Errorf("MOUSE_BUTTON expects 2 bytes, got %d", len(data))
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
			return Message{}, fmt.Errorf("MOUSE_SCROLL expects 1 byte, got %d", len(data))
		}
		return Message{
			Type: MsgMouseScroll,
			Payload: MouseScrollEvent{
				Delta: int8(data[0]),
			},
		}, nil

	case MsgACK:
		if len(data) != 1 {
			return Message{}, fmt.Errorf("ACK expects 1 byte, got %d", len(data))
		}
		return Message{
			Type: MsgACK,
			Payload: ACKEvent{
				Status: data[0],
			},
		}, nil

	case MsgPing:
		return Message{Type: MsgPing, Payload: PingEvent{}}, nil

	default:
		return Message{}, fmt.Errorf("unknown message type: 0x%02x", msgType)
	}
}
