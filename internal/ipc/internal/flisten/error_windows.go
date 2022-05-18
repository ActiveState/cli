package flisten

import (
	"errors"
	"io"
	"syscall"

	"github.com/ActiveState/cli/internal/errs"
	win "golang.org/x/sys/windows"
)

func asInUseError(err error) error {
	if errors.Is(err, win.WSAEADDRINUSE) {
		return ErrInUse
	}

	return err
}

func asConnRefusedError(err error) error {
	if errs.IsAny(err, win.WSAECONNREFUSED, win.WSAENETDOWN, win.WSAEINVAL) {
		return ErrConnRefused
	}
	return err
}

func asConnLostError(err error) error {
	if errs.IsAny(err, io.EOF, syscall.ECONNRESET, syscall.EPIPE, win.WSAECONNRESET, WSAENETRESET) {
		return ErrConnLost
	}
	return err
}
