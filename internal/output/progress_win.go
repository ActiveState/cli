//go:build windows
// +build windows

package output

import (
	"os"
	"syscall"
	"unsafe"
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
	_, _, _ = procGetConsoleScreenBufferInfo.Call(uintptr(handle), uintptr(unsafe.Pointer(&csbi)))

	var cursor coord
	cursor.x = csbi.cursorPosition.x + short(-n)
	cursor.y = csbi.cursorPosition.y

	_, _, _ = procSetConsoleCursorPosition.Call(uintptr(handle), uintptr(*(*int32)(unsafe.Pointer(&cursor))))
}
