package audio

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math"
	"os/exec"
	"strings"
	"sync"

	"github.com/ebitengine/oto/v3"
)

const (
	sampleRate   = 48000
	channelCount = 2
	bitDepth     = 2 // S16LE = 2 bytes per sample
)

// oto context singleton — oto.NewContext can only be called once per process.
var (
	otoOnce sync.Once
	otoCtx  *oto.Context
	otoErr  error
)

func getOtoContext() (*oto.Context, error) {
	otoOnce.Do(func() {
		op := &oto.NewContextOptions{
			SampleRate:   sampleRate,
			ChannelCount: channelCount,
			Format:       oto.FormatSignedInt16LE,
		}
		var readyCh chan struct{}
		otoCtx, readyCh, otoErr = oto.NewContext(op)
		if otoErr == nil {
			<-readyCh
		}
	})
	return otoCtx, otoErr
}

// PipelineConfig holds parameters for the audio capture pipeline.
type PipelineConfig struct {
	DeviceID string // platform-specific device identifier (empty = default)
}

// Pipeline manages a GStreamer audio capture subprocess and plays PCM via oto.
type Pipeline struct {
	mu      sync.Mutex
	running bool
	cmd     *exec.Cmd
	stopCh  chan struct{}

	volMu  sync.RWMutex
	volume float64 // 0.0–1.0
	muted  bool

	player *oto.Player
	pw     *io.PipeWriter
}

// NewPipeline creates a new audio pipeline (not yet started).
func NewPipeline() *Pipeline {
	return &Pipeline{
		volume: 1.0,
	}
}

// SetVolume sets the playback volume (0.0–1.0).
func (p *Pipeline) SetVolume(v float64) {
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	p.volMu.Lock()
	p.volume = v
	p.volMu.Unlock()
}

// SetMuted sets the mute state.
func (p *Pipeline) SetMuted(m bool) {
	p.volMu.Lock()
	p.muted = m
	p.volMu.Unlock()
}

// Start launches the GStreamer audio capture and oto playback.
func (p *Pipeline) Start(cfg PipelineConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return fmt.Errorf("audio pipeline already running")
	}

	// Build GStreamer pipeline args
	srcArgs := AudioSourceArgs(cfg.DeviceID)
	args := append([]string{"-e"}, srcArgs...)
	args = append(args, "!", "audioconvert", "!", "audioresample",
		"!", fmt.Sprintf("audio/x-raw,format=S16LE,rate=%d,channels=%d", sampleRate, channelCount),
		"!", "fdsink", "fd=1")

	cmdLine := "gst-launch-1.0 " + strings.Join(args, " ")
	log.Printf("audio: launching: %s", cmdLine)

	p.cmd = exec.Command("gst-launch-1.0", args...)
	hideWindow(p.cmd)

	stdout, err := p.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("audio stdout pipe: %w", err)
	}

	p.cmd.Stderr = log.Writer()

	if err := p.cmd.Start(); err != nil {
		return fmt.Errorf("starting audio gst-launch: %w", err)
	}

	// Get or create the singleton oto context
	ctx, err := getOtoContext()
	if err != nil {
		_ = p.cmd.Process.Kill()
		return fmt.Errorf("creating oto context: %w", err)
	}

	// io.Pipe connects the volume-scaled PCM writer to the oto player's reader
	pr, pw := io.Pipe()
	p.pw = pw
	p.player = ctx.NewPlayer(pr)
	p.player.Play()

	p.running = true
	p.stopCh = make(chan struct{})

	go p.readLoop(stdout)
	go p.waitForExit()

	log.Printf("audio: pipeline started (device-id=%q)", cfg.DeviceID)
	return nil
}

// Stop terminates the audio pipeline.
func (p *Pipeline) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return
	}

	close(p.stopCh)

	if p.cmd != nil && p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}

	if p.pw != nil {
		_ = p.pw.Close()
	}

	if p.player != nil {
		_ = p.player.Close()
	}

	p.running = false
	log.Printf("audio: pipeline stopped")
}

// IsRunning returns whether the audio pipeline is running.
func (p *Pipeline) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.running
}

// readLoop reads raw PCM from GStreamer stdout, applies volume scaling,
// and writes to the oto player via io.Pipe.
func (p *Pipeline) readLoop(r io.Reader) {
	// Read in chunks of 4096 bytes (must be multiple of 4 for S16LE stereo)
	buf := make([]byte, 4096)

	for {
		select {
		case <-p.stopCh:
			return
		default:
		}

		n, err := r.Read(buf)
		if err != nil {
			select {
			case <-p.stopCh:
				return
			default:
				if err != io.EOF {
					log.Printf("audio: read error: %v", err)
				}
				return
			}
		}

		if n == 0 {
			continue
		}

		// Apply volume scaling to PCM S16LE samples
		data := buf[:n]
		p.volMu.RLock()
		vol := p.volume
		muted := p.muted
		p.volMu.RUnlock()

		if muted {
			// Zero out the buffer
			for i := range data {
				data[i] = 0
			}
		} else if vol < 1.0 {
			// Scale each S16LE sample
			for i := 0; i+1 < len(data); i += 2 {
				sample := int16(binary.LittleEndian.Uint16(data[i : i+2]))
				scaled := float64(sample) * vol
				// Clamp
				if scaled > math.MaxInt16 {
					scaled = math.MaxInt16
				} else if scaled < math.MinInt16 {
					scaled = math.MinInt16
				}
				binary.LittleEndian.PutUint16(data[i:i+2], uint16(int16(scaled)))
			}
		}

		if _, err := p.pw.Write(data); err != nil {
			select {
			case <-p.stopCh:
				return
			default:
				log.Printf("audio: write error: %v", err)
				return
			}
		}
	}
}

func (p *Pipeline) waitForExit() {
	var exitErr error
	if p.cmd != nil {
		exitErr = p.cmd.Wait()
	}
	p.mu.Lock()
	p.running = false
	p.mu.Unlock()
	if exitErr != nil {
		log.Printf("audio: pipeline exited with error: %v", exitErr)
	} else {
		log.Printf("audio: pipeline exited")
	}
}
