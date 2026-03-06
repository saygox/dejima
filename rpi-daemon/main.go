package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"github.com/saygox/dejima/rpi-daemon/internal/injector"
	"github.com/saygox/dejima/rpi-daemon/internal/protocol"
	"github.com/saygox/dejima/rpi-daemon/internal/uart"
)

// Set at build time via: -ldflags "-X main.buildVersion=..."
var buildVersion = "dev"

func main() {
	showVersion := flag.Bool("version", false, "Print version and exit")
	showDiag := flag.Bool("diag", false, "Run diagnostics and exit")
	device := flag.String("device", "/dev/ttyAMA0", "UART device path")
	baud := flag.Int("baud", uart.DefaultBaudRate, "Baud rate")
	flag.Parse()

	if *showVersion {
		fmt.Println("dejima-kvm-daemon-rpi", buildVersion)
		os.Exit(0)
	}

	if *showDiag {
		fmt.Print(collectDiagnostics())
		os.Exit(0)
	}

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("dejima-kvm-daemon-rpi %s starting: device=%s baud=%d", buildVersion, *device, *baud)
	log.Printf("BUILD FINGERPRINT: go=%s os=%s arch=%s", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, s := range bi.Settings {
			if s.Key == "vcs.revision" || s.Key == "vcs.time" {
				log.Printf("BUILD FINGERPRINT: %s=%s", s.Key, s.Value)
			}
		}
	}

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
	stopCh := make(chan struct{})

	go func() {
		<-sigCh
		log.Printf("shutting down...")
		close(stopCh)
		inj.Close()
		port.Close()
		os.Exit(0)
	}()

	// Clipboard monitor: poll remote clipboard and send notifications on change
	go clipboardMonitor(port, inj, stopCh)

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

// clipboardMonitor polls the remote clipboard every 2s and sends
// MsgClipboardNotify over serial when the content changes.
func clipboardMonitor(port *uart.Port, inj *injector.Injector, stop <-chan struct{}) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	var lastContent string
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			text, err := inj.GetClipboard()
			if err != nil {
				log.Printf("clipboard monitor: read error: %v", err)
				continue
			}
			if text == lastContent {
				continue
			}
			truncated := text
			if len(truncated) > 50 {
				truncated = truncated[:50]
			}
			log.Printf("clipboard monitor: change detected, len=%d text=%q", len(text), truncated)
			lastContent = text
			if err := sendClipboardChunked(port, text, protocol.EncodeClipboardNotify); err != nil {
				log.Printf("clipboard notify send error: %v", err)
			}
		}
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

	case protocol.MouseAbsEvent:
		if err := inj.MouseAbsMove(int32(ev.X), int32(ev.Y)); err != nil {
			log.Printf("mouse abs inject error: %v", err)
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
		truncated := ev.Text
		if len(truncated) > 50 {
			truncated = truncated[:50]
		}
		log.Printf("text input: paste=%v final=%v len=%d text=%q", ev.Paste, ev.Final, len(ev.Text), truncated)
		if err := inj.TypeText(ev.Text, ev.Paste, ev.Final); err != nil {
			log.Printf("text input error: %v", err)
			sendACK(port, protocol.ACKError)
			return
		}
		sendACK(port, protocol.ACKOk)

	default:
		if msg.Type == protocol.MsgDiagReq {
			log.Printf("diag request received")
			text := collectDiagnostics()
			sendDiagChunked(port, text)
			return
		}
		if msg.Type == protocol.MsgClipboardReq {
			log.Printf("clipboard request received")
			text, err := inj.GetClipboard()
			if err != nil {
				log.Printf("clipboard read error: %v", err)
				sendACK(port, protocol.ACKError)
				return
			}
			log.Printf("clipboard response: %d bytes", len(text))
			if err := sendClipboardChunked(port, text, protocol.EncodeClipboardData); err != nil {
				log.Printf("clipboard send error: %v", err)
			}
			return
		}
		if msg.Type == protocol.MsgPing {
			_ = port.WriteFrame(protocol.EncodePing())
		}
	}
}

// sendClipboardChunked sends clipboard text in 199-byte UTF-8-safe chunks.
// encodeFn is either protocol.EncodeClipboardData or protocol.EncodeClipboardNotify.
func sendClipboardChunked(port *uart.Port, text string, encodeFn func(string, bool) []byte) error {
	const maxChunk = 199 // 1 byte msgType + 1 byte flags + up to 199 bytes text
	data := []byte(text)

	if len(data) == 0 {
		return port.WriteFrame(encodeFn("", true))
	}

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
			end = maxChunk
		}
		chunk := string(data[:end])
		data = data[end:]
		final := len(data) == 0

		if err := port.WriteFrame(encodeFn(chunk, final)); err != nil {
			return err
		}
	}
	return nil
}

func sendACK(port *uart.Port, status byte) {
	if err := port.WriteFrame(protocol.EncodeACK(status)); err != nil {
		log.Printf("failed to send ACK: %v", err)
	}
}

// ─── Diagnostics ──────────────────────────────────────────────

// sendDiagChunked sends diagnostic text in chunks over serial,
// followed by an empty chunk as end marker.
func sendDiagChunked(port *uart.Port, text string) {
	const maxChunk = 200
	data := []byte(text)
	for len(data) > 0 {
		end := maxChunk
		if end > len(data) {
			end = len(data)
		}
		if err := port.WriteFrame(protocol.EncodeDiagData(string(data[:end]))); err != nil {
			log.Printf("diag send error: %v", err)
			return
		}
		data = data[end:]
	}
	// End marker: empty diag data
	if err := port.WriteFrame(protocol.EncodeDiagData("")); err != nil {
		log.Printf("diag end marker send error: %v", err)
	}
}

// collectDiagnostics gathers system info and returns it as a string.
func collectDiagnostics() string {
	var b strings.Builder

	w := func(format string, args ...interface{}) {
		fmt.Fprintf(&b, format+"\n", args...)
	}

	w("=== dejima-kvm-daemon-rpi diagnostics ===")
	w("Version:  %s", buildVersion)
	w("Go:       %s %s/%s", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	w("")

	// OS info
	w("--- OS ---")
	appendCmd(&b, "cat", "/etc/os-release")
	appendCmd(&b, "uname", "-a")
	w("")

	// Display environment
	w("--- Display environment ---")
	for _, key := range []string{
		"DISPLAY", "WAYLAND_DISPLAY", "XDG_SESSION_TYPE",
		"XDG_RUNTIME_DIR", "XAUTHORITY", "DBUS_SESSION_BUS_ADDRESS",
	} {
		val := os.Getenv(key)
		if val == "" {
			w("  %-28s (not set)", key)
		} else {
			w("  %-28s %s", key, val)
		}
	}
	w("")

	// Wayland socket detection
	w("--- Wayland sockets ---")
	foundWayland := false
	for _, uid := range []string{"1000", "0"} {
		dir := "/run/user/" + uid
		matches, _ := filepath.Glob(dir + "/wayland-*")
		for _, m := range matches {
			if !strings.HasSuffix(m, ".lock") {
				w("  FOUND: %s", m)
				foundWayland = true
			}
		}
	}
	if !foundWayland {
		w("  (none found)")
	}
	w("")

	// Required tools
	w("--- Required tools ---")
	tools := []struct {
		name, backend string
	}{
		{"wtype", "wayland"},
		{"wl-paste", "wayland"},
		{"wl-copy", "wayland"},
		{"xdotool", "x11"},
		{"xclip", "x11"},
	}
	for _, t := range tools {
		path, err := exec.LookPath(t.name)
		if err != nil {
			w("  %-14s [%s]  NOT FOUND", t.name, t.backend)
		} else {
			w("  %-14s [%s]  OK  (%s)", t.name, t.backend, path)
		}
	}
	w("")

	// UART device
	w("--- UART ---")
	for _, dev := range []string{"/dev/ttyAMA0", "/dev/ttyS0", "/dev/serial0"} {
		if fi, err := os.Stat(dev); err == nil {
			w("  %-20s OK  (mode: %s)", dev, fi.Mode())
		} else {
			w("  %-20s not found", dev)
		}
	}
	w("")

	// uinput
	w("--- uinput ---")
	if fi, err := os.Stat("/dev/uinput"); err == nil {
		w("  /dev/uinput        OK  (mode: %s)", fi.Mode())
	} else {
		w("  /dev/uinput        NOT FOUND — sudo modprobe uinput")
	}

	return b.String()
}

func appendCmd(b *strings.Builder, name string, args ...string) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(b, "  error: %v\n", err)
		return
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		fmt.Fprintf(b, "  %s\n", line)
	}
}
