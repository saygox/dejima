package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/saygox/dejima/internal/audio"
	"github.com/saygox/dejima/internal/clipboard"
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
	audioWg    sync.WaitGroup // tracks async startAudio goroutine
	hid        *hid.Controller
	serialMu   sync.Mutex // protects serialPort and clipNotifyStopCh
	serialPort *serial.Port
	streamAddr string // "http://localhost:<port>" for MJPEG stream

	// Clipboard smart-paste state
	clipMu           sync.Mutex
	lastRemoteClip   string // last known remote clipboard content (from notify or sent)
	lastHostClipText string // last known host clipboard content (for WriteRemoteClipToHost guard)
	lastSentToRemote string // echo filter: text we just pasted to remote
	clipNotifyStopCh chan struct{}
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
	if text, err := clipboard.Read(); err == nil {
		a.lastHostClipText = text
	}
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

	var startErr error
	for attempt := 0; attempt < 3; attempt++ {
		a.pipeline = video.NewPipeline(a.store, video.PipelineConfig{
			DeviceIndex: a.cfg.DeviceIndex,
			DevicePath:  a.cfg.DevicePath,
			Width:       a.cfg.CaptureWidth,
			Height:      a.cfg.CaptureHeight,
			Quality:     a.cfg.JpegQuality,
		})
		if startErr = a.pipeline.Start(); startErr == nil {
			break
		}
		log.Printf("StartVideo: attempt %d failed: %v", attempt+1, startErr)
		if attempt < 2 {
			time.Sleep(500 * time.Millisecond)
		}
	}
	if startErr != nil {
		return startErr
	}

	// Start audio (best-effort, async so oto init doesn't block the UI)
	a.audioWg.Add(1)
	go func() {
		defer a.audioWg.Done()
		if err := a.startAudio(); err != nil {
			log.Printf("audio: failed to start: %v", err)
		}
	}()
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

func (a *App) startAudio() error {
	p := audio.NewPipeline()
	vol := float64(a.cfg.AudioVolume) / 100.0
	p.SetVolume(vol)
	p.SetMuted(a.cfg.AudioMuted)
	if err := p.Start(audio.PipelineConfig{
		DeviceID:   a.cfg.AudioDeviceID,
		SampleRate: a.cfg.AudioSampleRate,
	}); err != nil {
		return fmt.Errorf("audio: %w", err)
	}
	a.audioPipe = p
	return nil
}

func (a *App) stopAudio() {
	a.audioWg.Wait() // ensure async startAudio goroutine has finished
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

// SetAudioSampleRate sets the audio sample rate and saves config.
// Requires audio restart to take effect.
func (a *App) SetAudioSampleRate(rate int) {
	a.cfg.AudioSampleRate = rate
	_ = a.cfg.Save()
}

// GetAudioVolume returns the current audio volume (0-100).
func (a *App) GetAudioVolume() int {
	return a.cfg.AudioVolume
}

// GetAudioMuted returns the current audio mute state.
func (a *App) GetAudioMuted() bool {
	return a.cfg.AudioMuted
}

// GetAudioDiag returns detailed audio pipeline diagnostics.
func (a *App) GetAudioDiag() string {
	if a.audioPipe == nil {
		return "Audio pipeline: not running"
	}
	return a.audioPipe.Diag()
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
	a.disconnectSerialLocked()

	port, err := serial.Open(portName, a.cfg.BaudRate)
	if err != nil {
		return err
	}

	a.serialMu.Lock()
	a.serialPort = port
	a.clipNotifyStopCh = make(chan struct{})
	stopCh := a.clipNotifyStopCh
	a.serialMu.Unlock()

	a.hid.SetPort(port)

	// Start listening for remote clipboard notifications
	go a.remoteClipNotifyListener(port, stopCh)

	a.cfg.SerialPort = portName
	_ = a.cfg.Save()

	return nil
}

// DisconnectSerial closes the current serial connection.
func (a *App) DisconnectSerial() {
	a.disconnectSerialLocked()
}

func (a *App) disconnectSerialLocked() {
	a.serialMu.Lock()
	port := a.serialPort
	stopCh := a.clipNotifyStopCh
	a.serialPort = nil
	a.clipNotifyStopCh = nil
	a.serialMu.Unlock()

	if port != nil {
		if stopCh != nil {
			close(stopCh)
		}
		a.hid.SetPort(nil)
		_ = port.Close()

		// Reset remote clipboard so host-paste is preferred
		a.clipMu.Lock()
		a.lastRemoteClip = ""
		a.clipMu.Unlock()
	}
}

// GetSerialStatus returns the connected serial port name, or empty string.
func (a *App) GetSerialStatus() string {
	a.serialMu.Lock()
	defer a.serialMu.Unlock()
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
	truncated := text
	if len(truncated) > 50 {
		truncated = truncated[:50]
	}
	log.Printf("SendText: paste=%v len=%d text=%q", paste, len(text), truncated)
	err := a.hid.SendText(text, paste)
	if err != nil {
		log.Printf("SendText: error: %v", err)
	}
	return err
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
		log.Printf("GetRemoteClipboard: timed out after 3s")
		return "", fmt.Errorf("clipboard request timed out")
	}
}

// TestClipboardPipeline runs a diagnostic test of the clipboard pipeline
// and returns a multi-line result string.
func (a *App) TestClipboardPipeline() string {
	var lines []string
	w := func(format string, args ...interface{}) {
		lines = append(lines, fmt.Sprintf(format, args...))
	}

	w("=== Clipboard Pipeline Test ===")
	w("")

	// Step 1: Host clipboard read
	w("1. Host clipboard read (clipboard.Read)")
	hostText, err := clipboard.Read()
	if err != nil {
		w("   ERROR: %v", err)
	} else if hostText == "" {
		w("   OK (empty clipboard)")
	} else {
		preview := hostText
		if len(preview) > 60 {
			preview = preview[:60] + "..."
		}
		w("   OK: %d bytes — %q", len(hostText), preview)
	}
	w("")

	// Step 2: Serial connection
	w("2. Serial connection")
	if a.serialPort == nil {
		w("   ERROR: not connected")
		w("")
		w("Pipeline test stopped — serial required for remaining steps.")
		return joinLines(lines)
	}
	w("   OK: %s", a.serialPort.Name())
	w("")

	// Step 3: Send test text to RPi
	testStr := fmt.Sprintf("dejima-test-%s", time.Now().Format("150405"))
	w("3. Send test text to RPi (hid.SendText)")
	w("   Sending: %q", testStr)
	if err := a.hid.SendText(testStr, true); err != nil {
		w("   ERROR: %v", err)
	} else {
		w("   OK: sent successfully")
	}
	w("")

	// Step 4: Get remote clipboard
	w("4. Get remote clipboard (GetRemoteClipboard)")
	remoteText, err := a.GetRemoteClipboard()
	if err != nil {
		w("   ERROR: %v", err)
	} else if remoteText == "" {
		w("   OK (empty)")
	} else {
		preview := remoteText
		if len(preview) > 60 {
			preview = preview[:60] + "..."
		}
		w("   OK: %d bytes — %q", len(remoteText), preview)
	}

	return joinLines(lines)
}

func truncForLog(s string) string {
	if len(s) > 50 {
		return s[:50] + "..."
	}
	return s
}

func joinLines(lines []string) string {
	result := ""
	for i, l := range lines {
		if i > 0 {
			result += "\n"
		}
		result += l
	}
	return result
}

// --- Clipboard Smart Paste ---

// remoteClipNotifyListener reads ClipboardNotify messages from the serial port.
func (a *App) remoteClipNotifyListener(port *serial.Port, stop <-chan struct{}) {
	for {
		select {
		case <-stop:
			return
		case text := <-port.IncomingClipNotify:
			log.Printf("[clip-notify] received: len=%d text=%q", len(text), truncForLog(text))
			// Read host clipboard BEFORE taking lock (external command may be slow)
			hostClip, hostErr := clipboard.Read()

			a.clipMu.Lock()
			// Skip echo: if this matches what we just sent to remote, ignore it
			if text == a.lastSentToRemote {
				log.Printf("[clip-notify] echo filtered (matches lastSentToRemote)")
				a.lastSentToRemote = ""
				a.clipMu.Unlock()
				continue
			}
			a.lastRemoteClip = text
			// Sync lastHostClipText so stale diffs don't trigger
			// SendText on the next Cmd+V
			if hostErr == nil {
				log.Printf("[clip-notify] syncing lastHostClipText, len=%d", len(hostClip))
				a.lastHostClipText = hostClip
			}
			a.clipMu.Unlock()
		}
	}
}

// ResolveClipboardForPaste reads the host clipboard and decides whether to
// send the text to the remote. If the host clipboard changed since last check,
// the user copied something new on the host side — return it for SendText.
// If unchanged, return "" so the frontend sends a raw Ctrl+V instead,
// letting the remote use its own clipboard.
func (a *App) ResolveClipboardForPaste() string {
	hostClip, err := clipboard.Read()
	if err != nil {
		log.Printf("[resolve-paste] clipboard read error: %v", err)
		return ""
	}

	a.clipMu.Lock()
	defer a.clipMu.Unlock()

	if hostClip != a.lastHostClipText {
		// Host clipboard changed — user copied something new on host
		log.Printf("[resolve-paste] host changed → SendText, hostClip=%q lastHostClipText=%q",
			truncForLog(hostClip), truncForLog(a.lastHostClipText))
		a.lastHostClipText = hostClip
		return hostClip
	}

	// Host clipboard unchanged — let remote use its own clipboard (raw Ctrl+V)
	log.Printf("[resolve-paste] host unchanged → raw Ctrl+V")
	return ""
}

// MarkSentToRemote records text that was just pasted to the remote,
// so the echo notification from RPi can be filtered out.
func (a *App) MarkSentToRemote(text string) {
	a.clipMu.Lock()
	defer a.clipMu.Unlock()
	a.lastSentToRemote = text
	a.lastRemoteClip = text // remote now has this content too
}

// WriteRemoteClipToHost writes text to the host clipboard and tracks it
// so the next ResolveClipboardForPaste call doesn't treat it as a user-initiated change.
// If the user has changed the clipboard since the last known state, the write is skipped
// to avoid overwriting the user's content.
func (a *App) WriteRemoteClipToHost(text string) error {
	currentClip, readErr := clipboard.Read()

	a.clipMu.Lock()
	if readErr == nil && currentClip != a.lastHostClipText {
		// User has changed the clipboard since we last checked — don't overwrite
		a.lastHostClipText = currentClip
		a.clipMu.Unlock()
		return nil
	}
	a.lastRemoteClip = text
	a.lastHostClipText = text
	a.clipMu.Unlock()

	return clipboard.Write(text)
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
