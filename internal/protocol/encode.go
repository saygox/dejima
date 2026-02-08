package protocol

import "fmt"

// Encode serializes a Message into a payload byte slice (without framing).
// The first byte is always the message type ID.
func Encode(msg Message) ([]byte, error) {
	switch msg.Type {
	case MsgKeyEvent:
		ev, ok := msg.Payload.(KeyEvent)
		if !ok {
			return nil, fmt.Errorf("invalid payload for KEY_EVENT")
		}
		return []byte{
			MsgKeyEvent,
			byte(ev.Keycode >> 8),
			byte(ev.Keycode & 0xFF),
			ev.State,
		}, nil

	case MsgMouseMove:
		ev, ok := msg.Payload.(MouseMoveEvent)
		if !ok {
			return nil, fmt.Errorf("invalid payload for MOUSE_MOVE")
		}
		dx := uint16(ev.DX)
		dy := uint16(ev.DY)
		return []byte{
			MsgMouseMove,
			byte(dx >> 8),
			byte(dx & 0xFF),
			byte(dy >> 8),
			byte(dy & 0xFF),
		}, nil

	case MsgMouseButton:
		ev, ok := msg.Payload.(MouseButtonEvent)
		if !ok {
			return nil, fmt.Errorf("invalid payload for MOUSE_BUTTON")
		}
		return []byte{
			MsgMouseButton,
			ev.Button,
			ev.State,
		}, nil

	case MsgMouseScroll:
		ev, ok := msg.Payload.(MouseScrollEvent)
		if !ok {
			return nil, fmt.Errorf("invalid payload for MOUSE_SCROLL")
		}
		return []byte{
			MsgMouseScroll,
			byte(ev.Delta),
		}, nil

	case MsgMouseAbs:
		ev, ok := msg.Payload.(MouseAbsEvent)
		if !ok {
			return nil, fmt.Errorf("invalid payload for MOUSE_ABS")
		}
		return []byte{
			MsgMouseAbs,
			byte(ev.X >> 8),
			byte(ev.X & 0xFF),
			byte(ev.Y >> 8),
			byte(ev.Y & 0xFF),
		}, nil

	case MsgTextInput:
		ev, ok := msg.Payload.(TextInputEvent)
		if !ok {
			return nil, fmt.Errorf("invalid payload for TEXT_INPUT")
		}
		textBytes := []byte(ev.Text)
		buf := make([]byte, 1+len(textBytes))
		buf[0] = MsgTextInput
		copy(buf[1:], textBytes)
		return buf, nil

	case MsgClipboardReq:
		return []byte{MsgClipboardReq}, nil

	case MsgDiagReq:
		return []byte{MsgDiagReq}, nil

	case MsgACK:
		ev, ok := msg.Payload.(ACKEvent)
		if !ok {
			return nil, fmt.Errorf("invalid payload for ACK")
		}
		return []byte{
			MsgACK,
			ev.Status,
		}, nil

	case MsgPing:
		return []byte{MsgPing}, nil

	default:
		return nil, fmt.Errorf("unknown message type: 0x%02x", msg.Type)
	}
}
