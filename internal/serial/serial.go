package serial

import (
	"fmt"
	"log"
	"sync"

	"go.bug.st/serial"
)

// DefaultBaudRate is the UART baud rate for FT232 communication.
const DefaultBaudRate = 115200

// Port wraps a serial port connection with thread-safe write access.
type Port struct {
	port serial.Port
	mu   sync.Mutex
	name string
}

// Open opens a serial port with the given name and baud rate.
func Open(portName string, baudRate int) (*Port, error) {
	mode := &serial.Mode{
		BaudRate: baudRate,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	p, err := serial.Open(portName, mode)
	if err != nil {
		return nil, fmt.Errorf("opening serial port %s: %w", portName, err)
	}

	log.Printf("serial: opened %s at %d baud", portName, baudRate)

	return &Port{
		port: p,
		name: portName,
	}, nil
}

// Name returns the port name.
func (p *Port) Name() string {
	return p.name
}

// Write sends framed data over the serial port.
// The payload is wrapped with STX/LEN/CHECKSUM/ETX framing before sending.
func (p *Port) Write(payload []byte) error {
	frame, err := Frame(payload)
	if err != nil {
		return fmt.Errorf("framing: %w", err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	_, err = p.port.Write(frame)
	if err != nil {
		return fmt.Errorf("writing to serial: %w", err)
	}

	return nil
}

// Read reads one framed message from the serial port.
// Returns the payload (without framing).
func (p *Port) Read() ([]byte, error) {
	return ReadFrame(p.port)
}

// Close closes the serial port.
func (p *Port) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	log.Printf("serial: closing %s", p.name)
	return p.port.Close()
}

// ListPorts returns a list of available serial port names.
func ListPorts() ([]string, error) {
	ports, err := serial.GetPortsList()
	if err != nil {
		return nil, fmt.Errorf("listing serial ports: %w", err)
	}
	return ports, nil
}
