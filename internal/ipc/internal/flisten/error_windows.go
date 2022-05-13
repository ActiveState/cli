package flisten

import (
	"errors"

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

func errorIs(err error, errs ...error) bool {
	for _, e := range errs {
		if errors.Is(err, e) {
			return true
		}
	}
	return false
}
