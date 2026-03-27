package serial

import (
	"fmt"
	"log"
	"sync"
	"time"

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
	wg     sync.WaitGroup

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

	// pongCh receives a signal when a PingEvent (pong) arrives from RPi.
	pongCh chan struct{}

	// OnDead is called once when the connection is detected as dead
	// (e.g. Bluetooth out of range). Called from a new goroutine.
	OnDead   func()
	deadOnce sync.Once
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

	// Set read timeout so ReadFrame doesn't block forever when RPi is not responding.
	if err := p.SetReadTimeout(100 * time.Millisecond); err != nil {
		p.Close()
		return nil, fmt.Errorf("setting read timeout on %s: %w", portName, err)
	}

	log.Printf("serial: opened %s at %d baud", portName, baudRate)

	port := &Port{
		port:              p,
		name:              portName,
		stopCh:             make(chan struct{}),
		IncomingClipboard:  make(chan string, 1),
		IncomingClipNotify: make(chan string, 1),
		IncomingDiag:       make(chan string, 16),
		pongCh:             make(chan struct{}, 1),
	}

	port.wg.Add(1)
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
	defer p.wg.Done()
	const maxConsecErrors = 50 // ~5s at 100ms read timeout
	consecErrors := 0
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
				consecErrors++
				if consecErrors >= maxConsecErrors {
					log.Printf("serial: %d consecutive read errors, connection dead", consecErrors)
					p.deadOnce.Do(func() {
						if p.OnDead != nil {
							go p.OnDead()
						}
					})
					return
				}
				continue
			}
		}
		consecErrors = 0

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
		case protocol.PingEvent:
			select {
			case p.pongCh <- struct{}{}:
			default:
			}
		default:
			log.Printf("serial: unexpected message type 0x%02x", msg.Type)
		}
	}
}

// Ping sends a ping frame and waits for a pong response from the RPi daemon.
func (p *Port) Ping(timeout time.Duration) error {
	payload, err := protocol.Encode(protocol.Message{Type: protocol.MsgPing, Payload: protocol.PingEvent{}})
	if err != nil {
		return fmt.Errorf("encoding ping: %w", err)
	}

	// Write can block indefinitely if the remote side isn't draining the UART,
	// so run it in a goroutine with the same timeout.
	writeDone := make(chan error, 1)
	go func() {
		writeDone <- p.Write(payload)
	}()

	select {
	case err := <-writeDone:
		if err != nil {
			return fmt.Errorf("sending ping: %w", err)
		}
	case <-time.After(timeout):
		return fmt.Errorf("ping write timeout after %v", timeout)
	case <-p.stopCh:
		return fmt.Errorf("port closed")
	}

	select {
	case <-p.pongCh:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("ping timeout after %v", timeout)
	case <-p.stopCh:
		return fmt.Errorf("port closed")
	}
}

// Close closes the serial port.
func (p *Port) Close() error {
	close(p.stopCh)

	// Abort any pending write/read operations at the OS level.
	// On Windows this calls PurgeComm with PURGE_TXABORT which unblocks
	// a goroutine stuck in a blocking WriteFile/GetOverlappedResult.
	_ = p.port.ResetOutputBuffer()
	_ = p.port.ResetInputBuffer()

	log.Printf("serial: closing %s", p.name)
	err := p.port.Close()

	// Acquire the lock briefly so that any Write goroutine that just
	// woke up from the aborted write can release it cleanly.
	p.mu.Lock()
	p.mu.Unlock()

	// Wait for readLoop to finish so the goroutine doesn't outlive Close.
	p.wg.Wait()
	return err
}

// ListPorts returns a list of available serial port names.
func ListPorts() ([]string, error) {
	ports, err := serial.GetPortsList()
	if err != nil {
		return nil, fmt.Errorf("listing serial ports: %w", err)
	}
	return ports, nil
}
