package serial

import (
	"fmt"
	"log"
	"sync"

	"go.bug.st/serial"

	"github.com/saygox/dejima/internal/protocol"
)

// DefaultBaudRate is the UART baud rate for FT232 communication.
const DefaultBaudRate = 115200

// Port wraps a serial port connection with thread-safe write access
// and a background reader that dispatches incoming messages.
type Port struct {
	port   serial.Port
	mu     sync.Mutex
	name   string
	stopCh chan struct{}

	// Chunk accumulation buffers for multi-frame clipboard messages
	clipDataBuf   []byte
	clipNotifyBuf []byte

	// IncomingClipboard receives clipboard data sent from RPi (response to ClipboardReq).
	IncomingClipboard chan string
	// IncomingClipNotify receives unsolicited clipboard change notifications from RPi.
	IncomingClipNotify chan string
	// IncomingDiag receives diagnostic data chunks from RPi.
	// Empty string signals end of diagnostic output.
	IncomingDiag chan string
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

	port := &Port{
		port:              p,
		name:              portName,
		stopCh:             make(chan struct{}),
		IncomingClipboard:  make(chan string, 1),
		IncomingClipNotify: make(chan string, 1),
		IncomingDiag:       make(chan string, 16),
	}

	go port.readLoop()

	return port, nil
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

// readLoop reads framed messages from the serial port and dispatches them.
func (p *Port) readLoop() {
	for {
		select {
		case <-p.stopCh:
			return
		default:
		}

		payload, err := ReadFrame(p.port)
		if err != nil {
			select {
			case <-p.stopCh:
				return
			default:
				log.Printf("serial: read error: %v", err)
				continue
			}
		}

		msg, err := protocol.Decode(payload)
		if err != nil {
			log.Printf("serial: decode error: %v", err)
			continue
		}

		switch ev := msg.Payload.(type) {
		case protocol.ClipboardDataEvent:
			p.clipDataBuf = append(p.clipDataBuf, []byte(ev.Text)...)
			if ev.Final {
				text := string(p.clipDataBuf)
				p.clipDataBuf = nil
				// Non-blocking send to channel
				select {
				case p.IncomingClipboard <- text:
				default:
					select {
					case <-p.IncomingClipboard:
					default:
					}
					p.IncomingClipboard <- text
				}
			}
		case protocol.ClipboardNotifyEvent:
			p.clipNotifyBuf = append(p.clipNotifyBuf, []byte(ev.Text)...)
			if ev.Final {
				text := string(p.clipNotifyBuf)
				p.clipNotifyBuf = nil
				select {
				case p.IncomingClipNotify <- text:
				default:
					select {
					case <-p.IncomingClipNotify:
					default:
					}
					p.IncomingClipNotify <- text
				}
			}
		case protocol.DiagDataEvent:
			select {
			case p.IncomingDiag <- ev.Text:
			default:
				log.Printf("serial: diag channel full, dropping chunk")
			}
		case protocol.ACKEvent:
			// ACK received — currently unused, just log errors
			if ev.Status != 0 {
				log.Printf("serial: RPi returned ACK error")
			}
		default:
			log.Printf("serial: unexpected message type 0x%02x", msg.Type)
		}
	}
}

// Close closes the serial port.
func (p *Port) Close() error {
	close(p.stopCh)

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
