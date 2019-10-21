// The terminal mode manipulation code is derived heavily from:
// https://github.com/golang/crypto/blob/master/ssh/terminal/util.go:
// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package terminal

import (
	"bufio"
	"bytes"
	"fmt"
	"syscall"
	"unsafe"
)

type terminalRuneReaderState struct {
	in     FileReader
	term   syscall.Termios
	reader *bufio.Reader
	buf    *bytes.Buffer
}

func newTerminalRuneReaderState(input FileReader) RuneReaderState {
	buf := new(bytes.Buffer)
	return &terminalRuneReaderState{
		in: input,
		reader: bufio.NewReader(&BufferedReader{
			In:     input,
			Buffer: buf,
		}),
		buf: buf,
	}
}

func (s *terminalRuneReaderState) Buffer() *bytes.Buffer {
	return s.buf
}

// For reading runes we just want to disable echo.
func (s *terminalRuneReaderState) SetTermMode() error {
	if _, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(s.in.Fd()), ioctlReadTermios, uintptr(unsafe.Pointer(&s.term)), 0, 0, 0); err != 0 {
		return err
	}

	newState := s.term
	newState.Lflag &^= syscall.ECHO | syscall.ECHONL | syscall.ICANON | syscall.ISIG

	if _, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(s.in.Fd()), ioctlWriteTermios, uintptr(unsafe.Pointer(&newState)), 0, 0, 0); err != 0 {
		return err
	}

	return nil
}

func (s *terminalRuneReaderState) RestoreTermMode() error {
	if _, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(s.in.Fd()), ioctlWriteTermios, uintptr(unsafe.Pointer(&s.term)), 0, 0, 0); err != 0 {
		return err
	}
	return nil
}

func (s *terminalRuneReaderState) ReadRune() (rune, int, error) {
	r, size, err := s.reader.ReadRune()
	if err != nil {
		return r, size, err
	}

	// parse ^[ sequences to look for arrow keys
	if r == '\033' {
		if s.reader.Buffered() == 0 {
			// no more characters so must be `Esc` key
			return KeyEscape, 1, nil
		}
		r, size, err = s.reader.ReadRune()
		if err != nil {
			return r, size, err
		}
		if r != '[' {
			return r, size, fmt.Errorf("Unexpected Escape Sequence: %q", []rune{'\033', r})
		}
		r, size, err = s.reader.ReadRune()
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
			s.reader.Discard(1)
			return SpecialKeyDelete, 1, nil
		default:
			// discard the following '~' key from buffer
			s.reader.Discard(1)
			return IgnoreKey, 1, nil
		}
		return r, size, fmt.Errorf("Unknown Escape Sequence: %q", []rune{'\033', '[', r})
	}
	return r, size, err
}
