package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/saygox/dejima/internal/audio"
	"github.com/saygox/dejima/internal/config"
	"github.com/saygox/dejima/internal/hid"
	"github.com/saygox/dejima/internal/protocol"
	"github.com/saygox/dejima/internal/serial"
	"github.com/saygox/dejima/internal/video"
)

// App struct holds the application state and is bound to Wails.
type App struct {
	ctx        context.Context
	cfg        *config.Config
	store      *video.FrameStore
	pipeline   *video.Pipeline
	audioPipe  *audio.Pipeline
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
	a.stopAudio()
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
		DevicePath:  a.cfg.DevicePath,
		Width:       a.cfg.CaptureWidth,
		Height:      a.cfg.CaptureHeight,
		Quality:     a.cfg.JpegQuality,
	})
	if err := a.pipeline.Start(); err != nil {
		return err
	}

	// Start audio (best-effort, async so oto init doesn't block the UI)
	go a.startAudio()
	return nil
}

// StopVideo stops the video capture pipeline.
func (a *App) StopVideo() {
	log.Println("StopVideo: stopping audio…")
	a.stopAudio()
	log.Println("StopVideo: audio stopped, stopping video…")
	if a.pipeline != nil {
		_ = a.pipeline.Stop()
		a.pipeline = nil
	}
	log.Println("StopVideo: done")
}

// GetVideoStatus returns whether video is currently streaming.
func (a *App) GetVideoStatus() bool {
	return a.pipeline != nil && a.pipeline.IsRunning()
}

// GetVideoFrameCount returns the number of frames received (lightweight, for polling).
func (a *App) GetVideoFrameCount() int {
	if a.pipeline == nil {
		return 0
	}
	return int(a.pipeline.FrameCount())
}

// GetVideoDiag returns detailed video pipeline diagnostics.
func (a *App) GetVideoDiag() string {
	if a.pipeline == nil {
		return "Pipeline: not created\nDevice Index: " + fmt.Sprintf("%d", a.cfg.DeviceIndex) + "\nDevice Path: " + fmt.Sprintf("%q", a.cfg.DevicePath)
	}
	return a.pipeline.Diag()
}

// ListVideoDevices returns available video capture devices via GStreamer.
func (a *App) ListVideoDevices() ([]video.VideoDevice, error) {
	return video.ListDevices()
}

// SetDevice changes the video capture device by index and path.
// On Windows, path is used (ksvideosrc device-path); on macOS/Linux, index is used.
func (a *App) SetDevice(index int, path string) {
	a.cfg.DeviceIndex = index
	a.cfg.DevicePath = path
	_ = a.cfg.Save()
}

// SetCaptureResolution sets the capture width and height.
// Set both to 0 for auto-negotiation.
func (a *App) SetCaptureResolution(width, height int) {
	a.cfg.CaptureWidth = width
	a.cfg.CaptureHeight = height
	_ = a.cfg.Save()
}

// --- Audio ---

func (a *App) startAudio() {
	a.stopAudio()
	p := audio.NewPipeline()
	vol := float64(a.cfg.AudioVolume) / 100.0
	p.SetVolume(vol)
	p.SetMuted(a.cfg.AudioMuted)
	if err := p.Start(audio.PipelineConfig{
		DeviceID: a.cfg.AudioDeviceID,
	}); err != nil {
		log.Printf("audio: failed to start: %v", err)
		return
	}
	a.audioPipe = p
}

func (a *App) stopAudio() {
	if a.audioPipe != nil {
		log.Println("stopAudio: stopping audio pipeline…")
		a.audioPipe.Stop()
		a.audioPipe = nil
		log.Println("stopAudio: done")
	}
}

// ListAudioDevices returns available audio capture devices via GStreamer.
func (a *App) ListAudioDevices() ([]audio.AudioDevice, error) {
	return audio.ListDevices()
}

// SetAudioDevice changes the audio capture device ID and saves config.
func (a *App) SetAudioDevice(id string) {
	a.cfg.AudioDeviceID = id
	_ = a.cfg.Save()
}

// SetAudioVolume sets the audio volume (0-100) and saves config.
func (a *App) SetAudioVolume(level int) {
	if level < 0 {
		level = 0
	}
	if level > 100 {
		level = 100
	}
	a.cfg.AudioVolume = level
	_ = a.cfg.Save()
	if a.audioPipe != nil {
		a.audioPipe.SetVolume(float64(level) / 100.0)
	}
}

// SetAudioMuted sets the audio mute state and saves config.
func (a *App) SetAudioMuted(muted bool) {
	a.cfg.AudioMuted = muted
	_ = a.cfg.Save()
	if a.audioPipe != nil {
		a.audioPipe.SetMuted(muted)
	}
}

// GetAudioVolume returns the current audio volume (0-100).
func (a *App) GetAudioVolume() int {
	return a.cfg.AudioVolume
}

// GetAudioMuted returns the current audio mute state.
func (a *App) GetAudioMuted() bool {
	return a.cfg.AudioMuted
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
