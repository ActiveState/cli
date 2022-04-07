package flisten

import (
	"errors"

	"golang.org/x/sys/windows"
)

func asInUseError(err error) error {
	if errors.Is(err, windows.WSAEADDRINUSE) {
		return ErrInUse
	}

	return err
}

func asConnRefusedError(err error) error {
	if errors.Is(err, windows.WSAECONNREFUSED) || errors.Is(err, windows.WSAENETDOWN) {
		return ErrConnRefused
	}
	return err
}
