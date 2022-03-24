//go:build linux || darwin
// +build linux darwin

package flisten

import (
	"errors"
	"syscall"
)

func asInUse(err error) error {
	if errors.Is(err, syscall.EADDRINUSE) {
		return ErrInUse
	}
	return err
}

func asConnRefused(err error) error {
	if errors.Is(err, syscall.ECONNREFUSED) {
		return ErrConnRefused
	}
	return err
}
