package audio

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/saygox/dejima/internal/procgroup"
)

const (
	channelCount = 2
	bitDepth     = 2 // S16LE = 2 bytes per sample
)

// PipelineConfig holds parameters for the audio capture pipeline.
type PipelineConfig struct {
	DeviceID   string // platform-specific device identifier (empty = default)
	SampleRate int    // e.g. 44100, 48000 (0 defaults to 48000)
}

// Pipeline manages a single GStreamer process that captures audio and plays it
// directly (wasapi2src → volume → autoaudiosink). No PCM data passes through Go.
type Pipeline struct {
	mu           sync.Mutex
	running      bool
	cmd          *exec.Cmd
	stopCh       chan struct{}
	exitWg       sync.WaitGroup
	stderrBuf    bytes.Buffer
	cmdLine      string
	exitErr      string
	restartTimer *time.Timer

	volMu  sync.RWMutex
	volume float64 // 0.0–1.0
	muted  bool

	lastCfg PipelineConfig // saved for restart
}

// NewPipeline creates a new audio pipeline (not yet started).
func NewPipeline() *Pipeline {
	return &Pipeline{
		volume: 1.0,
	}
}

// SetVolume sets the playback volume (0.0–1.0).
// The pipeline is restarted with a debounce to apply the new volume.
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
	p.scheduleRestart()
}

// SetMuted sets the mute state.
// The pipeline is restarted with a debounce to apply the change.
func (p *Pipeline) SetMuted(m bool) {
	p.volMu.Lock()
	p.muted = m
	p.volMu.Unlock()
	p.scheduleRestart()
}

// Start launches a single GStreamer process: capture → volume → autoaudiosink.
func (p *Pipeline) Start(cfg PipelineConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return fmt.Errorf("audio pipeline already running")
	}

	rate := cfg.SampleRate
	if rate == 0 {
		rate = 48000
	}

	p.lastCfg = cfg

	p.volMu.RLock()
	vol := p.volume
	if p.muted {
		vol = 0.0
	}
	p.volMu.RUnlock()

	caps := fmt.Sprintf("audio/x-raw,format=S16LE,rate=%d,channels=%d", rate, channelCount)

	srcArgs := AudioSourceArgs(cfg.DeviceID)
	args := append([]string{"-e"}, srcArgs...)
	args = append(args, "!", "audioconvert", "!", "audioresample",
		"!", caps,
		"!", "volume", fmt.Sprintf("volume=%.2f", vol),
		"!", "autoaudiosink")

	p.cmdLine = "gst-launch-1.0 " + strings.Join(args, " ")
	log.Printf("audio: launching: %s", p.cmdLine)

	p.cmd = exec.Command("gst-launch-1.0", args...)
	hideWindow(p.cmd)

	p.stderrBuf.Reset()
	p.exitErr = ""
	p.cmd.Stderr = io.MultiWriter(log.Writer(), &p.stderrBuf)

	if err := p.cmd.Start(); err != nil {
		return fmt.Errorf("starting audio gst-launch: %w", err)
	}
	_ = procgroup.Add(p.cmd.Process.Pid)
	log.Printf("audio: gst-launch started (pid=%d)", p.cmd.Process.Pid)

	p.running = true
	p.stopCh = make(chan struct{})

	p.exitWg.Add(1)
	go p.waitForExit()

	log.Printf("audio: pipeline started (device-id=%q, volume=%.2f)", cfg.DeviceID, vol)
	return nil
}

// Stop terminates the audio pipeline.
func (p *Pipeline) Stop() {
	p.mu.Lock()

	if !p.running {
		p.mu.Unlock()
		return
	}

	if p.restartTimer != nil {
		p.restartTimer.Stop()
		p.restartTimer = nil
	}

	close(p.stopCh)

	if p.cmd != nil && p.cmd.Process != nil {
		_ = killProcess(p.cmd)
	}

	p.running = false
	p.mu.Unlock()

	p.exitWg.Wait() // wait for gst-launch to fully exit and release devices
	waitDeviceRelease()
	log.Printf("audio: pipeline stopped")
}

// IsRunning returns whether the audio pipeline is running.
func (p *Pipeline) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.running
}

// scheduleRestart debounces pipeline restarts when volume/mute changes.
func (p *Pipeline) scheduleRestart() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return
	}

	if p.restartTimer != nil {
		p.restartTimer.Stop()
	}
	p.restartTimer = time.AfterFunc(200*time.Millisecond, func() {
		p.Stop()
		if err := p.Start(p.lastCfg); err != nil {
			log.Printf("audio: restart failed: %v", err)
		}
	})
}

func (p *Pipeline) waitForExit() {
	defer p.exitWg.Done()
	exitErr := p.cmd.Wait()
	p.mu.Lock()
	p.running = false
	if exitErr != nil {
		p.exitErr = exitErr.Error()
	}
	stderrSnap := p.stderrBuf.String()
	p.mu.Unlock()
	if exitErr != nil {
		log.Printf("audio: pipeline exited with error: %v", exitErr)
	} else {
		log.Printf("audio: pipeline exited")
	}
	if len(stderrSnap) > 0 {
		log.Printf("audio: gstreamer stderr:\n%s", stderrSnap)
	}
}

// Diag returns diagnostic information about the audio pipeline.
func (p *Pipeline) Diag() string {
	p.mu.Lock()
	running := p.running
	stderr := p.stderrBuf.String()
	cmdLine := p.cmdLine
	exitErr := p.exitErr
	p.mu.Unlock()

	p.volMu.RLock()
	vol := p.volume
	muted := p.muted
	p.volMu.RUnlock()

	var b strings.Builder
	fmt.Fprintf(&b, "Running: %v\n", running)
	fmt.Fprintf(&b, "ExitErr: %s\n", exitErr)
	fmt.Fprintf(&b, "Volume: %.0f%% (muted=%v)\n", vol*100, muted)
	fmt.Fprintf(&b, "Pipeline: %s\n", cmdLine)
	fmt.Fprintf(&b, "\n--- GStreamer stderr ---\n")
	if stderr == "" {
		fmt.Fprintf(&b, "(empty)\n")
	} else {
		b.WriteString(stderr)
	}
	return b.String()
}
