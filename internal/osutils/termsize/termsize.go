//go:build linux

package termsize

import (
	"syscall"
	"unsafe"
)

type winsize struct {
	row, col       uint16
	xpixel, ypixel uint16
}

// GetTerminalColumns returns the number of columns available in the current terminal
func GetTerminalColumns() int {
	ws := winsize{}

	syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(0),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(&ws)))

	result := int(ws.col)
	if result == 0 {
		result = 80
	}
	return result
}
