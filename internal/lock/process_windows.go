//go:build windows

package lock

import "syscall"

const processQueryLimitedInformation = 0x1000

func processExists(pid int) bool {
	h, err := syscall.OpenProcess(processQueryLimitedInformation, false, uint32(pid))
	if err != nil {
		return false
	}
	syscall.CloseHandle(h)
	return true
}
