package video

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os/exec"
	"sync"
)

// VideoDevice represents a detected video capture device.
type VideoDevice struct {
	Index int    `json:"index"`
	Name  string `json:"name"`
	Path  string `json:"path"` // Windows ksvideosrc device-path
}

// ListDevices runs gst-device-monitor-1.0 to enumerate Video/Source devices.
func ListDevices() ([]VideoDevice, error) {
	cmd := exec.Command("gst-device-monitor-1.0", "Video/Source")
	hideWindow(cmd)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("running gst-device-monitor-1.0: %w", err)
	}
	return parseDeviceMonitorOutput(string(out)), nil
}

// parseDeviceMonitorOutput extracts device names and indices/paths from
// gst-device-monitor-1.0 output.
// macOS/Linux: "avfvideosrc device-index=0" / "v4l2src device-index=0"
// Windows:     "ksvideosrc device-path=\"\\\\?\\usb#...\""
func parseDeviceMonitorOutput(output string) []VideoDevice {
	var devices []VideoDevice
	lines := bytes.Split([]byte(output), []byte("\n"))

	var currentName string
	deviceCount := 0
	for _, line := range lines {
		s := string(bytes.TrimSpace(line))

		// "name  : UGREEN-25854"
		if bytes.HasPrefix(bytes.TrimSpace(line), []byte("name")) && bytes.Contains(line, []byte(":")) {
			parts := bytes.SplitN(line, []byte(":"), 2)
			if len(parts) == 2 {
				currentName = string(bytes.TrimSpace(parts[1]))
			}
			continue
		}

		// "gst-launch-1.0 avfvideosrc device-index=0 ! ..."
		if currentName != "" && bytes.Contains(line, []byte("device-index=")) {
			idx := bytes.Index(line, []byte("device-index="))
			if idx >= 0 {
				numStr := string(line[idx+len("device-index="):])
				var deviceIdx int
				for _, ch := range numStr {
					if ch >= '0' && ch <= '9' {
						deviceIdx = deviceIdx*10 + int(ch-'0')
					} else {
						break
					}
				}
				devices = append(devices, VideoDevice{
					Index: deviceIdx,
					Name:  currentName,
				})
			}
			currentName = ""
			deviceCount++
			continue
		}

		// "gst-launch-1.0 ksvideosrc device-path=\"\\?\usb#...\" ! ..."
		if currentName != "" && bytes.Contains(line, []byte("device-path=")) {
			idx := bytes.Index(line, []byte("device-path="))
			if idx >= 0 {
				rest := string(line[idx+len("device-path="):])
				path := extractQuotedValue(rest)
				devices = append(devices, VideoDevice{
					Index: deviceCount,
					Name:  currentName,
					Path:  path,
				})
			}
			currentName = ""
			deviceCount++
			continue
		}

		// Reset on next "Device found:"
		if s == "Device found:" {
			currentName = ""
		}
	}

	// Deduplicate by device name, preferring entries that have a Path (ksvideosrc).
	nameIdx := make(map[string]int, len(devices))
	var deduped []VideoDevice
	for _, d := range devices {
		if idx, ok := nameIdx[d.Name]; ok {
			// Replace if existing entry has no Path but new one does.
			if deduped[idx].Path == "" && d.Path != "" {
				deduped[idx] = d
			}
			continue
		}
		nameIdx[d.Name] = len(deduped)
		deduped = append(deduped, d)
	}

	return deduped
}

// extractQuotedValue extracts the value from a string like `"value" ! ...` or `value ! ...`.
func extractQuotedValue(s string) string {
	if len(s) == 0 {
		return ""
	}
	if s[0] == '"' {
		end := bytes.IndexByte([]byte(s[1:]), '"')
		if end >= 0 {
			return s[1 : end+1]
		}
		return s[1:]
	}
	// Unquoted: take until space or !
	for i, ch := range s {
		if ch == ' ' || ch == '!' {
			return s[:i]
		}
	}
	return s
}

// PipelineConfig holds parameters for building a GStreamer pipeline.
type PipelineConfig struct {
	DeviceIndex int
	DevicePath  string // Windows ksvideosrc device-path (takes priority if non-empty)
	Width       int    // 0 = auto
	Height      int    // 0 = auto
	Quality     int    // JPEG quality (1-100)
}

// Pipeline manages a GStreamer subprocess that produces JPEG frames.
type Pipeline struct {
	store  *FrameStore
	config PipelineConfig
	cmd    *exec.Cmd
	mu     sync.Mutex
	running bool
	stopCh  chan struct{}
}

// NewPipeline creates a new GStreamer pipeline controller.
func NewPipeline(store *FrameStore, cfg PipelineConfig) *Pipeline {
	if cfg.Quality <= 0 || cfg.Quality > 100 {
		cfg.Quality = 80
	}
	return &Pipeline{
		store:  store,
		config: cfg,
		stopCh: make(chan struct{}),
	}
}

// Start launches the GStreamer pipeline subprocess.
func (p *Pipeline) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return fmt.Errorf("pipeline already running")
	}

	args := p.buildPipelineArgs()
	cmdArgs := append([]string{"-q", "-e"}, args...)
	log.Printf("gstreamer: launching: gst-launch-1.0 %v", cmdArgs)

	p.cmd = exec.Command("gst-launch-1.0", cmdArgs...)
	hideWindow(p.cmd)

	stdout, err := p.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("creating stdout pipe: %w", err)
	}

	// Capture stderr for diagnostics
	p.cmd.Stderr = log.Writer()

	if err := p.cmd.Start(); err != nil {
		return fmt.Errorf("starting gst-launch: %w", err)
	}

	p.running = true
	p.stopCh = make(chan struct{})

	go p.readFrames(stdout)
	go p.waitForExit()

	log.Printf("gstreamer: pipeline started (device-index=%d, device-path=%q, %dx%d)",
		p.config.DeviceIndex, p.config.DevicePath, p.config.Width, p.config.Height)
	return nil
}

// buildPipelineArgs constructs gst-launch pipeline arguments as a slice.
// If Width/Height are set, a caps filter constrains the source resolution.
// If both are 0, no caps filter is added and GStreamer auto-negotiates.
func (p *Pipeline) buildPipelineArgs() []string {
	args := PipelineSourceArgs(p.config.DeviceIndex, p.config.DevicePath)

	if p.config.Width > 0 && p.config.Height > 0 {
		args = append(args, "!", fmt.Sprintf("video/x-raw,width=%d,height=%d", p.config.Width, p.config.Height))
	}

	args = append(args, "!", "videoconvert", "!", "jpegenc", fmt.Sprintf("quality=%d", p.config.Quality), "!", "fdsink", "fd=1")
	return args
}

// readFrames reads MJPEG multipart data from the gst-launch stdout.
func (p *Pipeline) readFrames(r io.Reader) {
	reader := bufio.NewReaderSize(r, 1024*1024)

	for {
		select {
		case <-p.stopCh:
			return
		default:
		}

		frame, err := readMultipartFrame(reader)
		if err != nil {
			select {
			case <-p.stopCh:
				return
			default:
				log.Printf("gstreamer: read error: %v", err)
				return
			}
		}

		if len(frame) > 0 {
			p.store.Update(frame)
		}
	}
}

// readMultipartFrame reads one JPEG frame from a multipart MJPEG stream.
// Scans for JPEG SOI (0xFFD8) and EOI (0xFFD9) markers.
func readMultipartFrame(r *bufio.Reader) ([]byte, error) {
	// Scan for JPEG SOI marker (0xFF 0xD8)
	for {
		b, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		if b == 0xFF {
			next, err := r.ReadByte()
			if err != nil {
				return nil, err
			}
			if next == 0xD8 {
				// Found SOI
				break
			}
		}
	}

	// Accumulate bytes until we find EOI (0xFF 0xD9)
	var buf bytes.Buffer
	buf.Write([]byte{0xFF, 0xD8})

	for {
		b, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		buf.WriteByte(b)

		if b == 0xFF {
			next, err := r.ReadByte()
			if err != nil {
				return nil, err
			}
			buf.WriteByte(next)
			if next == 0xD9 {
				// Found EOI - complete frame
				return buf.Bytes(), nil
			}
		}
	}
}

func (p *Pipeline) waitForExit() {
	if p.cmd != nil {
		_ = p.cmd.Wait()
	}
	p.mu.Lock()
	p.running = false
	p.mu.Unlock()
	log.Printf("gstreamer: pipeline exited")
}

// Stop terminates the GStreamer pipeline.
func (p *Pipeline) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return nil
	}

	close(p.stopCh)

	if p.cmd != nil && p.cmd.Process != nil {
		if err := p.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("killing gst-launch: %w", err)
		}
	}

	p.running = false
	log.Printf("gstreamer: pipeline stopped")
	return nil
}

// IsRunning returns whether the pipeline is currently running.
func (p *Pipeline) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.running
}
