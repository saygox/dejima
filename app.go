package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/saygox/kvm-like/internal/config"
	"github.com/saygox/kvm-like/internal/hid"
	"github.com/saygox/kvm-like/internal/protocol"
	"github.com/saygox/kvm-like/internal/serial"
	"github.com/saygox/kvm-like/internal/video"
)

// App struct holds the application state and is bound to Wails.
type App struct {
	ctx        context.Context
	cfg        *config.Config
	store      *video.FrameStore
	pipeline   *video.Pipeline
	hid        *hid.Controller
	serialPort *serial.Port
	streamAddr string // "http://localhost:<port>" for MJPEG stream
}

// NewApp creates a new App instance.
func NewApp() *App {
	cfg, err := config.Load()
	if err != nil {
		log.Printf("config: using defaults: %v", err)
		cfg = config.DefaultConfig()
	}

	store := video.NewFrameStore()

	return &App{
		cfg:   cfg,
		store: store,
		hid:   hid.NewController(),
	}
}

// startup is called when the Wails app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.startStreamServer()
}

// shutdown is called when the Wails app is closing.
func (a *App) shutdown(ctx context.Context) {
	a.StopVideo()
	a.DisconnectSerial()
}

// startStreamServer launches a standalone HTTP server for MJPEG streaming.
// Wails AssetServer does not support http.Flusher, so streaming responses
// (like MJPEG multipart/x-mixed-replace) must be served separately.
func (a *App) startStreamServer() {
	mjpeg := video.NewMJPEGHandler(a.store)

	mux := http.NewServeMux()
	mux.Handle("/stream", mjpeg)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Printf("stream server: failed to listen: %v", err)
		return
	}

	port := listener.Addr().(*net.TCPAddr).Port
	a.streamAddr = fmt.Sprintf("http://127.0.0.1:%d", port)
	log.Printf("stream server: listening on %s/stream", a.streamAddr)

	go func() {
		if err := http.Serve(listener, mux); err != nil {
			log.Printf("stream server: %v", err)
		}
	}()
}

// GetStreamURL returns the MJPEG stream URL.
func (a *App) GetStreamURL() string {
	return a.streamAddr + "/stream"
}

// --- Video ---

// StartVideo starts the GStreamer video capture pipeline.
func (a *App) StartVideo() error {
	if a.pipeline != nil && a.pipeline.IsRunning() {
		return nil
	}

	a.pipeline = video.NewPipeline(a.store, video.PipelineConfig{
		DeviceIndex: a.cfg.DeviceIndex,
		Width:       a.cfg.CaptureWidth,
		Height:      a.cfg.CaptureHeight,
		Quality:     a.cfg.JpegQuality,
	})
	if err := a.pipeline.Start(); err != nil {
		return err
	}
	return nil
}

// StopVideo stops the video capture pipeline.
func (a *App) StopVideo() {
	if a.pipeline != nil {
		_ = a.pipeline.Stop()
		a.pipeline = nil
	}
}

// GetVideoStatus returns whether video is currently streaming.
func (a *App) GetVideoStatus() bool {
	return a.pipeline != nil && a.pipeline.IsRunning()
}

// ListVideoDevices returns available video capture devices via GStreamer.
func (a *App) ListVideoDevices() ([]video.VideoDevice, error) {
	return video.ListDevices()
}

// SetDeviceIndex changes the video capture device.
func (a *App) SetDeviceIndex(index int) {
	a.cfg.DeviceIndex = index
	_ = a.cfg.Save()
}

// SetCaptureResolution sets the capture width and height.
// Set both to 0 for auto-negotiation.
func (a *App) SetCaptureResolution(width, height int) {
	a.cfg.CaptureWidth = width
	a.cfg.CaptureHeight = height
	_ = a.cfg.Save()
}

// --- Serial ---

// ListSerialPorts returns available serial port names.
func (a *App) ListSerialPorts() ([]string, error) {
	return serial.ListPorts()
}

// DetectFT232 attempts to auto-detect the FT232 serial port.
func (a *App) DetectFT232() (string, error) {
	return serial.DetectFT232()
}

// ConnectSerial opens the given serial port.
func (a *App) ConnectSerial(portName string) error {
	a.DisconnectSerial()

	port, err := serial.Open(portName, a.cfg.BaudRate)
	if err != nil {
		return err
	}

	a.serialPort = port
	a.hid.SetPort(port)

	a.cfg.SerialPort = portName
	_ = a.cfg.Save()

	return nil
}

// DisconnectSerial closes the current serial connection.
func (a *App) DisconnectSerial() {
	if a.serialPort != nil {
		a.hid.SetPort(nil)
		_ = a.serialPort.Close()
		a.serialPort = nil
	}
}

// GetSerialStatus returns the connected serial port name, or empty string.
func (a *App) GetSerialStatus() string {
	if a.serialPort != nil {
		return a.serialPort.Name()
	}
	return ""
}

// --- HID Input ---

// SendKeyEvent sends a keyboard event to RPi via serial.
func (a *App) SendKeyEvent(jsCode string, pressed bool) error {
	return a.hid.SendKeyEvent(jsCode, pressed)
}

// SendMouseMove sends a relative mouse movement event.
func (a *App) SendMouseMove(dx, dy int) error {
	return a.hid.SendMouseMove(dx, dy)
}

// SendMouseButton sends a mouse button event.
func (a *App) SendMouseButton(jsButton int, pressed bool) error {
	return a.hid.SendMouseButton(jsButton, pressed)
}

// SendMouseAbs sends an absolute mouse position (0-32767 normalized).
func (a *App) SendMouseAbs(x, y int) error {
	return a.hid.SendMouseAbs(x, y)
}

// SendMouseScroll sends a mouse scroll event.
func (a *App) SendMouseScroll(delta int) error {
	return a.hid.SendMouseScroll(delta)
}

// SendText sends a UTF-8 text string to the remote machine.
// paste=false uses wtype/xdotool (for terminals), paste=true uses wl-copy+Ctrl+V (for browsers).
func (a *App) SendText(text string, paste bool) error {
	return a.hid.SendText(text, paste)
}

// GetRemoteClipboard requests clipboard text from the RPi and returns it.
func (a *App) GetRemoteClipboard() (string, error) {
	if a.serialPort == nil {
		return "", fmt.Errorf("serial not connected")
	}

	// Send clipboard request
	payload, err := protocol.Encode(protocol.Message{
		Type:    protocol.MsgClipboardReq,
		Payload: nil,
	})
	if err != nil {
		return "", fmt.Errorf("encoding clipboard request: %w", err)
	}
	if err := a.serialPort.Write(payload); err != nil {
		return "", fmt.Errorf("sending clipboard request: %w", err)
	}

	// Wait for response with timeout
	select {
	case text := <-a.serialPort.IncomingClipboard:
		return text, nil
	case <-time.After(3 * time.Second):
		return "", fmt.Errorf("clipboard request timed out")
	}
}

// --- Diagnostics ---

// GetRemoteDiag requests diagnostic info from the RPi daemon and returns it as text.
func (a *App) GetRemoteDiag() (string, error) {
	if a.serialPort == nil {
		return "", fmt.Errorf("serial not connected")
	}

	// Drain any stale diag data
	for {
		select {
		case <-a.serialPort.IncomingDiag:
		default:
			goto drained
		}
	}
drained:

	payload, err := protocol.Encode(protocol.Message{
		Type: protocol.MsgDiagReq, Payload: nil,
	})
	if err != nil {
		return "", fmt.Errorf("encoding diag request: %w", err)
	}
	if err := a.serialPort.Write(payload); err != nil {
		return "", fmt.Errorf("sending diag request: %w", err)
	}

	// Collect chunks until empty string (end marker) or timeout
	var buf []byte
	timeout := time.After(5 * time.Second)
	for {
		select {
		case chunk := <-a.serialPort.IncomingDiag:
			if chunk == "" {
				return string(buf), nil
			}
			buf = append(buf, chunk...)
		case <-timeout:
			if len(buf) > 0 {
				return string(buf), nil
			}
			return "", fmt.Errorf("diagnostic request timed out")
		}
	}
}

// --- Config ---

// GetConfig returns the current configuration.
func (a *App) GetConfig() *config.Config {
	return a.cfg
}

// SaveConfig saves the current configuration.
func (a *App) SaveConfig() error {
	return a.cfg.Save()
}
