package injector

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"syscall"
	"time"
)

// uinput ioctl commands
const (
	_uiSetEvBit   uintptr = 0x40045564
	_uiSetKeyBit  uintptr = 0x40045565
	_uiSetAbsBit  uintptr = 0x40045567
	_uiSetPropBit uintptr = 0x4004556e
	_uiDevCreate  uintptr = 0x5501
	_uiDevDestroy uintptr = 0x5502
)

// input event types and codes
const (
	_evSyn uint16 = 0x00
	_evKey uint16 = 0x01
	_evAbs uint16 = 0x03

	_absX uint16 = 0x00
	_absY uint16 = 0x01

	_synReport uint16 = 0

	_btnLeft  uintptr = 0x110
	_btnRight uintptr = 0x111

	_inputPropDirect uintptr = 0x01

	_uinputMaxNameSize = 80
	_absSize           = 64
	_busUSB            = 0x03
)

type _inputID struct {
	Bustype uint16
	Vendor  uint16
	Product uint16
	Version uint16
}

type _uinputUserDev struct {
	Name       [_uinputMaxNameSize]byte
	ID         _inputID
	EffectsMax uint32
	Absmax     [_absSize]int32
	Absmin     [_absSize]int32
	Absfuzz    [_absSize]int32
	Absflat    [_absSize]int32
}

type _inputEvent struct {
	Time  syscall.Timeval
	Type  uint16
	Code  uint16
	Value int32
}

// absPointer is a uinput device with INPUT_PROP_DIRECT that provides
// absolute cursor positioning. Unlike a touchpad (which libinput converts
// to relative movements), libinput treats a device with INPUT_PROP_DIRECT
// as a direct input device and maps ABS_X/ABS_Y to screen coordinates.
type absPointer struct {
	fd *os.File
}

func newAbsPointer(name string, maxX, maxY int32) (*absPointer, error) {
	fd, err := os.OpenFile("/dev/uinput", syscall.O_WRONLY|syscall.O_NONBLOCK, 0660)
	if err != nil {
		return nil, fmt.Errorf("open /dev/uinput: %w", err)
	}

	// Set INPUT_PROP_DIRECT so libinput treats this as a direct input device
	// (absolute coordinates map to screen positions, not relative deltas).
	if err := uinputIoctl(fd, _uiSetPropBit, _inputPropDirect); err != nil {
		fd.Close()
		return nil, fmt.Errorf("set INPUT_PROP_DIRECT: %w", err)
	}

	// Register EV_ABS + ABS_X + ABS_Y
	if err := uinputIoctl(fd, _uiSetEvBit, uintptr(_evAbs)); err != nil {
		fd.Close()
		return nil, fmt.Errorf("set EV_ABS: %w", err)
	}
	for _, axis := range []uintptr{uintptr(_absX), uintptr(_absY)} {
		if err := uinputIoctl(fd, _uiSetAbsBit, axis); err != nil {
			fd.Close()
			return nil, fmt.Errorf("set ABS axis %d: %w", axis, err)
		}
	}

	// Register EV_KEY + BTN_LEFT + BTN_RIGHT (needed for pointer recognition)
	if err := uinputIoctl(fd, _uiSetEvBit, uintptr(_evKey)); err != nil {
		fd.Close()
		return nil, fmt.Errorf("set EV_KEY: %w", err)
	}
	for _, btn := range []uintptr{_btnLeft, _btnRight} {
		if err := uinputIoctl(fd, _uiSetKeyBit, btn); err != nil {
			fd.Close()
			return nil, fmt.Errorf("set BTN %d: %w", btn, err)
		}
	}

	// Build the uinput_user_dev struct
	var devName [_uinputMaxNameSize]byte
	copy(devName[:], name)

	var absMax [_absSize]int32
	absMax[_absX] = maxX
	absMax[_absY] = maxY

	dev := _uinputUserDev{
		Name: devName,
		ID: _inputID{
			Bustype: _busUSB,
			Vendor:  0x1209, // pid.codes VID for open-source projects
			Product: 0x4b56, // "KV" in ASCII
			Version: 1,
		},
		Absmax: absMax,
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, dev); err != nil {
		fd.Close()
		return nil, fmt.Errorf("encode uinput_user_dev: %w", err)
	}
	if _, err := fd.Write(buf.Bytes()); err != nil {
		fd.Close()
		return nil, fmt.Errorf("write uinput_user_dev: %w", err)
	}

	if err := uinputIoctl(fd, _uiDevCreate, 0); err != nil {
		fd.Close()
		return nil, fmt.Errorf("UI_DEV_CREATE: %w", err)
	}

	time.Sleep(200 * time.Millisecond)
	return &absPointer{fd: fd}, nil
}

// MoveTo sends absolute positioning events to move the cursor.
func (p *absPointer) MoveTo(x, y int32) error {
	events := [3]_inputEvent{
		{Type: _evAbs, Code: _absX, Value: x},
		{Type: _evAbs, Code: _absY, Value: y},
		{Type: _evSyn, Code: _synReport, Value: 0},
	}
	for _, ev := range events {
		buf := new(bytes.Buffer)
		if err := binary.Write(buf, binary.LittleEndian, ev); err != nil {
			return fmt.Errorf("encode input_event: %w", err)
		}
		if _, err := p.fd.Write(buf.Bytes()); err != nil {
			return fmt.Errorf("write input_event: %w", err)
		}
	}
	return nil
}

// Close destroys the uinput device and closes the file descriptor.
func (p *absPointer) Close() error {
	if p.fd == nil {
		return nil
	}
	_ = uinputIoctl(p.fd, _uiDevDestroy, 0)
	return p.fd.Close()
}

func uinputIoctl(fd *os.File, cmd, arg uintptr) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd.Fd(), cmd, arg)
	if errno != 0 {
		return errno
	}
	return nil
}
