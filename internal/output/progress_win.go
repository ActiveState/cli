//go:build windows
// +build windows

package output

import (
	"os"
	"syscall"
	"unsafe"

	"github.com/ActiveState/cli/internal/multilog"
)

var kernel32 = syscall.NewLazyDLL("kernel32.dll")
var procGetConsoleScreenBufferInfo = kernel32.NewProc("GetConsoleScreenBufferInfo")
var procSetConsoleCursorPosition = kernel32.NewProc("SetConsoleCursorPosition")

type coord struct {
	x short
	y short
}

type short int16
type word uint16

type smallRect struct {
	bottom short
	left   short
	right  short
	top    short
}

type consoleScreenBufferInfo struct {
	size              coord
	cursorPosition    coord
	attributes        word
	window            smallRect
	maximumWindowSize coord
}

func (d *Spinner) moveCaretBackInCommandPrompt(n int) {
	handle := syscall.Handle(os.Stdout.Fd())

	var csbi consoleScreenBufferInfo
	if _, _, err := procGetConsoleScreenBufferInfo.Call(uintptr(handle), uintptr(unsafe.Pointer(&csbi))); err != nil {
		var cursor coord
		cursor.x = csbi.cursorPosition.x + short(-n)
		cursor.y = csbi.cursorPosition.y

		_, _, err2 := procSetConsoleCursorPosition.Call(uintptr(handle), uintptr(*(*int32)(unsafe.Pointer(&cursor))))
		if err2 != nil && !d.reportedError {
			multilog.Error("Error calling SetConsoleCursorPosition: %v", err2)
			d.reportedError = true
		}
	} else if !d.reportedError {
		multilog.Error("Error calling GetConsoleScreenBufferInfo: %v", err)
		d.reportedError = true
	}
}
