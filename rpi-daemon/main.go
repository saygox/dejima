package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/saygox/kvm-like/rpi-daemon/internal/injector"
	"github.com/saygox/kvm-like/rpi-daemon/internal/protocol"
	"github.com/saygox/kvm-like/rpi-daemon/internal/uart"
)

func main() {
	device := flag.String("device", "/dev/ttyAMA0", "UART device path")
	baud := flag.Int("baud", uart.DefaultBaudRate, "Baud rate")
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("kvm-daemon starting: device=%s baud=%d", *device, *baud)

	// Open UART
	port, err := uart.Open(*device, *baud)
	if err != nil {
		log.Fatalf("failed to open UART: %v", err)
	}
	defer port.Close()

	// Create uinput injector
	inj, err := injector.New()
	if err != nil {
		log.Fatalf("failed to create injector: %v", err)
	}
	defer inj.Close()

	// Handle shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Printf("shutting down...")
		inj.Close()
		port.Close()
		os.Exit(0)
	}()

	// Main loop: read frames and inject events
	log.Printf("listening for HID events...")
	for {
		payload, err := port.ReadFrame()
		if err != nil {
			log.Printf("read error: %v", err)
			continue
		}

		msg, err := protocol.Decode(payload)
		if err != nil {
			log.Printf("decode error: %v", err)
			sendACK(port, protocol.ACKError)
			continue
		}

		handleMessage(port, inj, msg)
	}
}

func handleMessage(port *uart.Port, inj *injector.Injector, msg protocol.Message) {
	switch ev := msg.Payload.(type) {
	case protocol.KeyEvent:
		var err error
		if ev.State == protocol.StatePress {
			err = inj.KeyPress(int(ev.Keycode))
		} else {
			err = inj.KeyRelease(int(ev.Keycode))
		}
		if err != nil {
			log.Printf("key inject error: %v", err)
			sendACK(port, protocol.ACKError)
			return
		}
		sendACK(port, protocol.ACKOk)

	case protocol.MouseMoveEvent:
		if err := inj.MouseMove(int32(ev.DX), int32(ev.DY)); err != nil {
			log.Printf("mouse move inject error: %v", err)
			sendACK(port, protocol.ACKError)
			return
		}
		sendACK(port, protocol.ACKOk)

	case protocol.MouseButtonEvent:
		var err error
		if ev.State == protocol.StatePress {
			err = inj.MouseButtonPress(ev.Button)
		} else {
			err = inj.MouseButtonRelease(ev.Button)
		}
		if err != nil {
			log.Printf("mouse button inject error: %v", err)
			sendACK(port, protocol.ACKError)
			return
		}
		sendACK(port, protocol.ACKOk)

	case protocol.MouseScrollEvent:
		if err := inj.MouseScroll(int32(ev.Delta)); err != nil {
			log.Printf("mouse scroll inject error: %v", err)
			sendACK(port, protocol.ACKError)
			return
		}
		sendACK(port, protocol.ACKOk)

	case protocol.TextInputEvent:
		if err := inj.TypeText(ev.Text); err != nil {
			log.Printf("text input error: %v", err)
			sendACK(port, protocol.ACKError)
			return
		}
		sendACK(port, protocol.ACKOk)

	default:
		if msg.Type == protocol.MsgPing {
			_ = port.WriteFrame(protocol.EncodePing())
		}
	}
}

func sendACK(port *uart.Port, status byte) {
	if err := port.WriteFrame(protocol.EncodeACK(status)); err != nil {
		log.Printf("failed to send ACK: %v", err)
	}
}
