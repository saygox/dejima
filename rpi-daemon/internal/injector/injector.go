package injector

import (
	"fmt"
	"log"
	"os/exec"

	"github.com/bendahl/uinput"
)

// Injector manages virtual keyboard and mouse devices via uinput.
type Injector struct {
	keyboard uinput.Keyboard
	mouse    uinput.Mouse
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

	log.Printf("injector: virtual devices created")
	return &Injector{keyboard: kb, mouse: mouse}, nil
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

// MouseScroll sends a vertical scroll event.
func (inj *Injector) MouseScroll(delta int32) error {
	return inj.mouse.Wheel(false, delta)
}

// TypeText types a UTF-8 string using xdotool.
// This supports Japanese and other non-ASCII characters.
func (inj *Injector) TypeText(text string) error {
	cmd := exec.Command("xdotool", "type", "--clearmodifiers", "--", text)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("xdotool type: %w: %s", err, out)
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
	log.Printf("injector: virtual devices closed")
}
