package injector

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"

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
	keyboard    uinput.Keyboard
	mouse       uinput.Mouse
	touchpad    uinput.TouchPad
	backend     displayBackend
	env         []string // environment for subprocess calls
	sessionUser string   // desktop session user (e.g. "pi") for running wtype/wl-paste
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

	backend, env, sessionUser := detectBackend()
	log.Printf("injector: virtual devices created (keyboard, mouse, touchpad), backend=%s, session_user=%s", backendName(backend), sessionUser)

	return &Injector{keyboard: kb, mouse: mouse, touchpad: tp, backend: backend, env: env, sessionUser: sessionUser}, nil
}

func backendName(b displayBackend) string {
	if b == backendWayland {
		return "wayland"
	}
	return "x11"
}

// detectBackend checks if Wayland or X11 is running and builds the
// environment variables needed for subprocess calls (wtype, wl-paste, etc.).
// Also returns the username that owns the desktop session.
func detectBackend() (displayBackend, []string, string) {
	env := os.Environ()
	// Ensure UTF-8 locale for subprocess tools (wtype, xdotool, etc.)
	// systemd services have no locale by default, causing multi-byte corruption.
	env = ensureEnv(env, "LANG", "C.UTF-8")

	// Check if WAYLAND_DISPLAY is already set
	waylandDisplay := envGet(env, "WAYLAND_DISPLAY")
	if waylandDisplay != "" {
		env = ensureEnv(env, "WAYLAND_DISPLAY", waylandDisplay)
		env = ensureXDGRuntime(env)
		sessionUser := detectSessionUser(envGet(env, "XDG_RUNTIME_DIR"))
		log.Printf("injector: detected wayland (WAYLAND_DISPLAY=%s)", waylandDisplay)
		return backendWayland, env, sessionUser
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
				sessionUser := detectSessionUser(dir)
				log.Printf("injector: detected wayland socket %s in %s (user=%s)", base, dir, sessionUser)
				return backendWayland, env, sessionUser
			}
		}
	}

	// Fallback to X11
	env = ensureEnv(env, "DISPLAY", ":0")
	if envGet(env, "XAUTHORITY") == "" {
		// Find .Xauthority in any user's home directory
		if xauth := findXauthority(); xauth != "" {
			env = append(env, "XAUTHORITY="+xauth)
		}
	}
	log.Printf("injector: falling back to X11")
	return backendX11, env, ""
}

// detectSessionUser finds the username from XDG_RUNTIME_DIR path like /run/user/1000.
func detectSessionUser(xdgDir string) string {
	// Extract UID from /run/user/<uid>
	parts := strings.Split(xdgDir, "/")
	for i, p := range parts {
		if p == "user" && i+1 < len(parts) {
			uid := parts[i+1]
			if u, err := user.LookupId(uid); err == nil {
				return u.Username
			}
		}
	}
	return ""
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

// findXauthority searches for a .Xauthority file in home directories.
func findXauthority() string {
	// Check /root first
	if _, err := os.Stat("/root/.Xauthority"); err == nil {
		return "/root/.Xauthority"
	}
	// Scan /home/*/
	matches, _ := filepath.Glob("/home/*/.Xauthority")
	if len(matches) > 0 {
		return matches[0]
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

// sessionCmd builds a command that runs as the desktop session user.
// When the daemon runs as root but the Wayland compositor belongs to
// another user (e.g. pi), we use "runuser" to switch.
func (inj *Injector) sessionCmd(name string, args ...string) *exec.Cmd {
	var cmd *exec.Cmd
	if inj.sessionUser != "" && isRoot() {
		// Run as the session user so it can connect to the Wayland compositor
		allArgs := append([]string{"-u", inj.sessionUser, "--", name}, args...)
		cmd = exec.Command("runuser", allArgs...)
	} else {
		cmd = exec.Command(name, args...)
	}
	cmd.Env = inj.env
	return cmd
}

func isRoot() bool {
	return os.Geteuid() == 0
}

// GetClipboard reads the current clipboard content.
// Wayland: wl-paste, X11: xclip
func (inj *Injector) GetClipboard() (string, error) {
	var cmd *exec.Cmd
	if inj.backend == backendWayland {
		cmd = inj.sessionCmd("wl-paste", "--no-newline")
	} else {
		cmd = inj.sessionCmd("xclip", "-selection", "clipboard", "-o")
	}
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("clipboard read (%s): %w", backendName(inj.backend), err)
	}
	return string(out), nil
}

// TypeText sends a UTF-8 string to the focused application.
// paste=false (Type mode): uses wtype (Wayland) or xdotool (X11) to simulate keystrokes.
// paste=true (Paste mode): sets clipboard with wl-copy/xclip, then sends Ctrl+V via uinput.
func (inj *Injector) TypeText(text string, paste bool) error {
	if paste {
		return inj.typeTextPaste(text)
	}
	return inj.typeTextDirect(text)
}

// typeTextDirect types text using wtype (Wayland) or xdotool (X11).
// Works in terminals but may be ignored by browser URL bars.
func (inj *Injector) typeTextDirect(text string) error {
	var cmd *exec.Cmd
	if inj.backend == backendWayland {
		cmd = inj.sessionCmd("wtype", "--", text)
	} else {
		cmd = inj.sessionCmd("xdotool", "type", "--clearmodifiers", "--", text)
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("type text (%s): %w", backendName(inj.backend), err)
	}
	return nil
}

// typeTextPaste types text via clipboard paste (wl-copy/xclip + Ctrl+V).
// Works in browsers but not in terminals where Ctrl+V has different meaning.
func (inj *Injector) typeTextPaste(text string) error {
	var cmd *exec.Cmd
	if inj.backend == backendWayland {
		cmd = inj.sessionCmd("wl-copy", "--", text)
	} else {
		cmd = inj.sessionCmd("xclip", "-selection", "clipboard")
		cmd.Stdin = strings.NewReader(text)
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("set clipboard (%s): %w", backendName(inj.backend), err)
	}

	time.Sleep(50 * time.Millisecond)
	inj.keyboard.KeyDown(uinput.KeyLeftctrl)
	inj.keyboard.KeyDown(uinput.KeyV)
	time.Sleep(30 * time.Millisecond)
	inj.keyboard.KeyUp(uinput.KeyV)
	inj.keyboard.KeyUp(uinput.KeyLeftctrl)
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
