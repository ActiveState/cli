package terminal

import (
	"bufio"
	"bytes"
	"fmt"
	"syscall"
	"unsafe"
)

var (
	dll              = syscall.NewLazyDLL("kernel32.dll")
	setConsoleMode   = dll.NewProc("SetConsoleMode")
	getConsoleMode   = dll.NewProc("GetConsoleMode")
	readConsoleInput = dll.NewProc("ReadConsoleInputW")
)

const (
	EVENT_KEY = 0x0001

	// key codes for arrow keys
	// https://msdn.microsoft.com/en-us/library/windows/desktop/dd375731(v=vs.85).aspx
	VK_DELETE = 0x2E
	VK_END    = 0x23
	VK_HOME   = 0x24
	VK_LEFT   = 0x25
	VK_UP     = 0x26
	VK_RIGHT  = 0x27
	VK_DOWN   = 0x28

	RIGHT_CTRL_PRESSED = 0x0004
	LEFT_CTRL_PRESSED  = 0x0008

	ENABLE_ECHO_INPUT      uint32 = 0x0004
	ENABLE_LINE_INPUT      uint32 = 0x0002
	ENABLE_PROCESSED_INPUT uint32 = 0x0001
)

type inputRecord struct {
	eventType uint16
	padding   uint16
	event     [16]byte
}

type keyEventRecord struct {
	bKeyDown          int32
	wRepeatCount      uint16
	wVirtualKeyCode   uint16
	wVirtualScanCode  uint16
	unicodeChar       uint16
	wdControlKeyState uint32
}

type runeReaderState struct {
	term uint32
	buf  *bufio.Reader
}

func newRuneReaderState(input FileReader) runeReaderState {
	return runeReaderState{
		buf: bufio.NewReader(input),
	}
}

func (rr *RuneReader) Buffer() *bytes.Buffer {
	return nil
}

func (rr *RuneReader) SetTermMode() error {
	r, _, err := getConsoleMode.Call(uintptr(rr.stdio.In.Fd()), uintptr(unsafe.Pointer(&rr.state.term)))
	// windows return 0 on error
	if r == 0 {
		return err
	}

	newState := rr.state.term
	newState &^= ENABLE_ECHO_INPUT | ENABLE_LINE_INPUT | ENABLE_PROCESSED_INPUT
	r, _, err = setConsoleMode.Call(uintptr(rr.stdio.In.Fd()), uintptr(newState))
	// windows return 0 on error
	if r == 0 {
		return err
	}
	return nil
}

func (rr *RuneReader) RestoreTermMode() error {
	r, _, err := setConsoleMode.Call(uintptr(rr.stdio.In.Fd()), uintptr(rr.state.term))
	// windows return 0 on error
	if r == 0 {
		return err
	}
	return nil
}

func (rr *RuneReader) ReadRune() (rune, int, error) {
	r, size, err := rr.state.buf.ReadRune()
	if err != nil {
		return r, size, err
	}

	// parse ^[ sequences to look for arrow keys
	if r == '\033' {
		if rr.state.buf.Buffered() == 0 {
			// no more characters so must be `Esc` key
			return KeyEscape, 1, nil
		}
		r, size, err = rr.state.buf.ReadRune()
		if err != nil {
			return r, size, err
		}
		if r != '[' {
			return r, size, fmt.Errorf("Unexpected Escape Sequence: %q", []rune{'\033', r})
		}
		r, size, err = rr.state.buf.ReadRune()
		if err != nil {
			return r, size, err
		}
		switch r {
		case 'D':
			return KeyArrowLeft, 1, nil
		case 'C':
			return KeyArrowRight, 1, nil
		case 'A':
			return KeyArrowUp, 1, nil
		case 'B':
			return KeyArrowDown, 1, nil
		case 'H': // Home button
			return SpecialKeyHome, 1, nil
		case 'F': // End button
			return SpecialKeyEnd, 1, nil
		case '3': // Delete Button
			// discard the following '~' key from buffer
			rr.state.buf.Discard(1)
			return SpecialKeyDelete, 1, nil
		default:
			// discard the following '~' key from buffer
			rr.state.buf.Discard(1)
			return IgnoreKey, 1, nil
		}
		return r, size, fmt.Errorf("Unknown Escape Sequence: %q", []rune{'\033', '[', r})
	}
	return r, size, err
}
