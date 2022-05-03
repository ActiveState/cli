package flisten

import (
	"errors"

	win "golang.org/x/sys/windows"
)

func asInUseError(err error) error {
	if errors.Is(err, windows.WSAEADDRINUSE) {
		return ErrInUse
	}

	return err
}

func asConnRefusedError(err error) error {
	if errorIs(err, win.WSAECONNREFUSED, win.WSAENETDOWN, win.WSAEINVAL) {
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
