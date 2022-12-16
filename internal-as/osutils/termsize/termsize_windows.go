// +build windows

package termsize

import (
	"syscall"

	"github.com/Azure/go-ansiterm/winterm"
)

func getStdHandle(stdhandle int) (uintptr, error) {
	handle, err := syscall.GetStdHandle(stdhandle)
	if err != nil {
		return 0, err
	}
	return uintptr(handle), nil
}

// GetTerminalColumns returns the number of columns available in the current terminal
func GetTerminalColumns() int {
	defaultWidth := 80

	stdoutHandle, err := getStdHandle(syscall.STD_OUTPUT_HANDLE)
	if err != nil {
		return defaultWidth
	}

	info, err := winterm.GetConsoleScreenBufferInfo(stdoutHandle)
	if err != nil {
		return defaultWidth
	}

	if info.MaximumWindowSize.X > 0 {
		return int(info.MaximumWindowSize.X)
	}

	return defaultWidth
}
