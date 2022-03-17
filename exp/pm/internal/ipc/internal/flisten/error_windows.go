package flisten

import (
	"errors"

	"golang.org/x/sys/windows"
)

func asInUse(err error) error {
	if errors.Is(err, windows.WSAEADDRINUSE) {
		return ErrInUse
	}

	return err
}

func asConnRefused(err error) error {
	if errors.Is(err, windows.WSAECONNREFUSED) {
		return ErrConnRefused
	}
	return err
}
