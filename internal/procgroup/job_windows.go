//go:build windows

package procgroup

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	jobObjectExtendedLimitInformation = 9
	jobObjectLimitKillOnJobClose      = 0x2000
)

var job windows.Handle

// Init creates a Windows Job Object with KILL_ON_JOB_CLOSE, so all child
// processes are automatically terminated when the parent process exits
// (including crashes and forced kills).
func Init() error {
	h, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return err
	}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
		BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
			LimitFlags: jobObjectLimitKillOnJobClose,
		},
	}
	_, err = windows.SetInformationJobObject(h, jobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info)))
	if err != nil {
		windows.CloseHandle(h)
		return err
	}
	job = h
	return nil
}

// Add assigns a running process to the Job Object so it will be killed
// when the parent process exits.
func Add(pid int) error {
	if job == 0 {
		return nil
	}
	h, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		return err
	}
	defer windows.CloseHandle(h)
	return windows.AssignProcessToJobObject(job, h)
}
