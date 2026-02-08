package uart

import (
	"fmt"
	"io"
	"log"
	"sync"

	"go.bug.st/serial"
)

const (
	STX            byte = 0x02
	ETX            byte = 0x03
	MaxPayloadSize      = 256
	DefaultBaudRate     = 115200
)

// Port wraps a serial port for the UART listener.
type Port struct {
	port serial.Port
	mu   sync.Mutex
	name string
}

// Open opens the specified UART device.
func Open(devicePath string, baudRate int) (*Port, error) {
	mode := &serial.Mode{
		BaudRate: baudRate,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	p, err := serial.Open(devicePath, mode)
	if err != nil {
		return nil, fmt.Errorf("opening UART %s: %w", devicePath, err)
	}

	log.Printf("uart: opened %s at %d baud", devicePath, baudRate)
	return &Port{port: p, name: devicePath}, nil
}

// ReadFrame reads one framed message. Returns the payload.
func (p *Port) ReadFrame() ([]byte, error) {
	return readFrame(p.port)
}

// WriteFrame sends a framed message.
func (p *Port) WriteFrame(payload []byte) error {
	frame, err := frame(payload)
	if err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	_, err = p.port.Write(frame)
	return err
}

// Close closes the port.
func (p *Port) Close() error {
	return p.port.Close()
}

func readFrame(r io.Reader) ([]byte, error) {
	buf := make([]byte, 1)
	for {
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		if buf[0] == STX {
			break
		}
	}

	lenBuf := make([]byte, 2)
	if _, err := io.ReadFull(r, lenBuf); err != nil {
		return nil, fmt.Errorf("reading length: %w", err)
	}
	length := int(lenBuf[0])<<8 | int(lenBuf[1])

	if length == 0 || length > MaxPayloadSize {
		return nil, fmt.Errorf("invalid payload length: %d", length)
	}

	rest := make([]byte, length+2)
	if _, err := io.ReadFull(r, rest); err != nil {
		return nil, fmt.Errorf("reading payload: %w", err)
	}

	payload := rest[:length]
	checksum := rest[length]
	etx := rest[length+1]

	if etx != ETX {
		return nil, fmt.Errorf("expected ETX, got 0x%02x", etx)
	}

	if xorChecksum(payload) != checksum {
		return nil, fmt.Errorf("checksum mismatch")
	}

	return payload, nil
}

func frame(payload []byte) ([]byte, error) {
	n := len(payload)
	if n == 0 || n > MaxPayloadSize {
		return nil, fmt.Errorf("invalid payload length: %d", n)
	}

	checksum := xorChecksum(payload)
	f := make([]byte, 0, n+5)
	f = append(f, STX)
	f = append(f, byte(n>>8), byte(n&0xFF))
	f = append(f, payload...)
	f = append(f, checksum)
	f = append(f, ETX)
	return f, nil
}

func xorChecksum(data []byte) byte {
	var cs byte
	for _, b := range data {
		cs ^= b
	}
	return cs
}
