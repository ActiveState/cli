//go:build linux || darwin
// +build linux darwin

package flisten

import (
	"errors"
	"syscall"
)

func asInUseError(err error) error {
	if errors.Is(err, syscall.EADDRINUSE) {
		return ErrInUse
	}
	return err
}

func asConnRefusedError(err error) error {
	if errors.Is(err, syscall.ECONNREFUSED) {
		return ErrConnRefused
	}
	return err
}
