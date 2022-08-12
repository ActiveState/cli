//go:build linux || darwin
// +build linux darwin

package flisten

import (
	"errors"
	"io"
	"syscall"

	"github.com/ActiveState/cli/internal/errs"
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

func asConnLostError(err error) error {
	if errs.IsAny(err, io.EOF, syscall.ECONNRESET, syscall.EPIPE) {
		return ErrConnLost
	}
	return err
}
