//go:build !windows && !plan9 && !solaris && !appengine && !wasm
// +build !windows,!plan9,!solaris,!appengine,!wasm

package termsize

import (
	"syscall"
	"unsafe"

	"github.com/ActiveState/cli/internal/multilog"
)

const (
	termSizeFallback = 80
)

type winsize struct {
	row, col       uint16
	xpixel, ypixel uint16
}

// GetTerminalColumns returns the number of columns available in the current terminal
func GetTerminalColumns() int {
	ws := winsize{}

	_, _, err := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(0),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(&ws)))
	if err != 0 {
		multilog.Error("Error getting terminal size: %v", err)
		return termSizeFallback
	}

	result := int(ws.col)
	if result == 0 {
		result = termSizeFallback
	}
	return result
}
