package serial

import (
	"fmt"
	"io"
)

// Framing constants
const (
	STX byte = 0x02
	ETX byte = 0x03
)

// MaxPayloadSize is the maximum payload length.
const MaxPayloadSize = 256

// Frame wraps a payload with STX/LEN/CHECKSUM/ETX framing.
// Format: [STX][LEN_HI][LEN_LO][PAYLOAD...][CHECKSUM_XOR][ETX]
func Frame(payload []byte) ([]byte, error) {
	n := len(payload)
	if n == 0 {
		return nil, fmt.Errorf("empty payload")
	}
	if n > MaxPayloadSize {
		return nil, fmt.Errorf("payload too large: %d > %d", n, MaxPayloadSize)
	}

	checksum := xorChecksum(payload)

	// STX + LEN(2) + PAYLOAD(n) + CHECKSUM(1) + ETX = n + 5
	frame := make([]byte, 0, n+5)
	frame = append(frame, STX)
	frame = append(frame, byte(n>>8), byte(n&0xFF))
	frame = append(frame, payload...)
	frame = append(frame, checksum)
	frame = append(frame, ETX)

	return frame, nil
}

// ReadFrame reads one framed message from the reader.
// Returns the payload (without framing).
func ReadFrame(r io.Reader) ([]byte, error) {
	// Read until STX
	buf := make([]byte, 1)
	for {
		_, err := io.ReadFull(r, buf)
		if err != nil {
			return nil, fmt.Errorf("reading STX: %w", err)
		}
		if buf[0] == STX {
			break
		}
	}

	// Read length (2 bytes)
	lenBuf := make([]byte, 2)
	if _, err := io.ReadFull(r, lenBuf); err != nil {
		return nil, fmt.Errorf("reading length: %w", err)
	}
	length := int(lenBuf[0])<<8 | int(lenBuf[1])

	if length == 0 || length > MaxPayloadSize {
		return nil, fmt.Errorf("invalid payload length: %d", length)
	}

	// Read payload + checksum + ETX
	rest := make([]byte, length+2)
	if _, err := io.ReadFull(r, rest); err != nil {
		return nil, fmt.Errorf("reading payload: %w", err)
	}

	payload := rest[:length]
	checksum := rest[length]
	etx := rest[length+1]

	if etx != ETX {
		return nil, fmt.Errorf("expected ETX (0x03), got 0x%02x", etx)
	}

	if xorChecksum(payload) != checksum {
		return nil, fmt.Errorf("checksum mismatch")
	}

	return payload, nil
}

func xorChecksum(data []byte) byte {
	var cs byte
	for _, b := range data {
		cs ^= b
	}
	return cs
}
