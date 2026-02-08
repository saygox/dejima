package injector

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bendahl/uinput"
)

// AbsMax is the maximum value for absolute positioning (0 to AbsMax).
const AbsMax = 32767

// displayBackend describes whether we talk to Wayland or X11.
type displayBackend int

const (
	backendX11 displayBackend = iota
	backendWayland
)

// Injector manages virtual keyboard and mouse devices via uinput.
type Injector struct {
	keyboard uinput.Keyboard
	mouse    uinput.Mouse
	touchpad uinput.TouchPad
	backend  displayBackend
	env      []string // environment for subprocess calls
}

// New creates a new uinput Injector.
// Requires /dev/uinput access (typically root or uinput group).
func New() (*Injector, error) {
	kb, err := uinput.CreateKeyboard("/dev/uinput", []byte("kvm-keyboard"))
	if err != nil {
		return nil, fmt.Errorf("creating virtual keyboard: %w", err)
	}

	mouse, err := uinput.CreateMouse("/dev/uinput", []byte("kvm-mouse"))
	if err != nil {
		kb.Close()
		return nil, fmt.Errorf("creating virtual mouse: %w", err)
	}

	tp, err := uinput.CreateTouchPad("/dev/uinput", []byte("kvm-touchpad"), 0, 0, AbsMax, AbsMax)
	if err != nil {
		kb.Close()
		mouse.Close()
		return nil, fmt.Errorf("creating virtual touchpad: %w", err)
	}

	backend, env := detectBackend()
	log.Printf("injector: virtual devices created (keyboard, mouse, touchpad), backend=%s", backendName(backend))

	return &Injector{keyboard: kb, mouse: mouse, touchpad: tp, backend: backend, env: env}, nil
}

func backendName(b displayBackend) string {
	if b == backendWayland {
		return "wayland"
	}
	return "x11"
}

// detectBackend checks if Wayland or X11 is running and builds the
// environment variables needed for subprocess calls (wtype, wl-paste, etc.).
func detectBackend() (displayBackend, []string) {
	env := os.Environ()

	// Check if WAYLAND_DISPLAY is already set
	waylandDisplay := envGet(env, "WAYLAND_DISPLAY")
	if waylandDisplay != "" {
		env = ensureEnv(env, "WAYLAND_DISPLAY", waylandDisplay)
		env = ensureXDGRuntime(env)
		log.Printf("injector: detected wayland (WAYLAND_DISPLAY=%s)", waylandDisplay)
		return backendWayland, env
	}

	// Probe: look for wayland socket in common XDG_RUNTIME_DIR locations
	for _, uid := range []string{"1000", "0"} {
		dir := "/run/user/" + uid
		matches, _ := filepath.Glob(dir + "/wayland-*")
		for _, m := range matches {
			base := filepath.Base(m)
			if strings.HasPrefix(base, "wayland-") && !strings.HasSuffix(base, ".lock") {
				env = ensureEnv(env, "WAYLAND_DISPLAY", base)
				env = ensureEnv(env, "XDG_RUNTIME_DIR", dir)
				log.Printf("injector: detected wayland socket %s in %s", base, dir)
				return backendWayland, env
			}
		}
	}

	// Fallback to X11
	env = ensureEnv(env, "DISPLAY", ":0")
	// XAUTHORITY for root running under systemd
	if envGet(env, "XAUTHORITY") == "" {
		for _, path := range []string{"/home/pi/.Xauthority", "/root/.Xauthority"} {
			if _, err := os.Stat(path); err == nil {
				env = append(env, "XAUTHORITY="+path)
				break
			}
		}
	}
	log.Printf("injector: falling back to X11")
	return backendX11, env
}

func ensureXDGRuntime(env []string) []string {
	if envGet(env, "XDG_RUNTIME_DIR") != "" {
		return env
	}
	// Try common paths
	for _, uid := range []string{"1000", "0"} {
		dir := "/run/user/" + uid
		if fi, err := os.Stat(dir); err == nil && fi.IsDir() {
			return append(env, "XDG_RUNTIME_DIR="+dir)
		}
	}
	return env
}

func envGet(env []string, key string) string {
	prefix := key + "="
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			return e[len(prefix):]
		}
	}
	return ""
}

func ensureEnv(env []string, key, value string) []string {
	prefix := key + "="
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			return env // already set
		}
	}
	return append(env, prefix+value)
}

// KeyPress sends a key press event.
func (inj *Injector) KeyPress(keycode int) error {
	return inj.keyboard.KeyDown(keycode)
}

// KeyRelease sends a key release event.
func (inj *Injector) KeyRelease(keycode int) error {
	return inj.keyboard.KeyUp(keycode)
}

// MouseMove sends a relative mouse movement.
func (inj *Injector) MouseMove(dx, dy int32) error {
	return inj.mouse.Move(dx, dy)
}

// MouseButtonPress presses a mouse button (1=left, 2=right, 3=middle).
func (inj *Injector) MouseButtonPress(button byte) error {
	switch button {
	case 1:
		return inj.mouse.LeftPress()
	case 2:
		return inj.mouse.RightPress()
	case 3:
		return inj.mouse.MiddlePress()
	default:
		return inj.mouse.LeftPress()
	}
}

// MouseButtonRelease releases a mouse button.
func (inj *Injector) MouseButtonRelease(button byte) error {
	switch button {
	case 1:
		return inj.mouse.LeftRelease()
	case 2:
		return inj.mouse.RightRelease()
	case 3:
		return inj.mouse.MiddleRelease()
	default:
		return inj.mouse.LeftRelease()
	}
}

// MouseAbsMove moves the cursor to an absolute position (0-32767 normalized).
func (inj *Injector) MouseAbsMove(x, y int32) error {
	return inj.touchpad.MoveTo(x, y)
}

// MouseScroll sends a vertical scroll event.
func (inj *Injector) MouseScroll(delta int32) error {
	return inj.mouse.Wheel(false, delta)
}

// GetClipboard reads the current clipboard content.
// Wayland: wl-paste, X11: xclip
func (inj *Injector) GetClipboard() (string, error) {
	var cmd *exec.Cmd
	if inj.backend == backendWayland {
		cmd = exec.Command("wl-paste", "--no-newline")
	} else {
		cmd = exec.Command("xclip", "-selection", "clipboard", "-o")
	}
	cmd.Env = inj.env
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("clipboard read (%s): %w", backendName(inj.backend), err)
	}
	return string(out), nil
}

// TypeText types a UTF-8 string.
// Wayland: wtype, X11: xdotool
func (inj *Injector) TypeText(text string) error {
	var cmd *exec.Cmd
	if inj.backend == backendWayland {
		cmd = exec.Command("wtype", "--", text)
	} else {
		cmd = exec.Command("xdotool", "type", "--clearmodifiers", "--", text)
	}
	cmd.Env = inj.env
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("type text (%s): %w: %s", backendName(inj.backend), err, out)
	}
	return nil
}

// Close destroys the virtual devices.
func (inj *Injector) Close() {
	if inj.keyboard != nil {
		inj.keyboard.Close()
	}
	if inj.mouse != nil {
		inj.mouse.Close()
	}
	if inj.touchpad != nil {
		inj.touchpad.Close()
	}
	log.Printf("injector: virtual devices closed")
}
