package flisten

import (
	"errors"
	"syscall"
)

var (
	ErrInUse        = errors.New("flisten in use")
	ErrConnRefused  = errors.New("flisten connection refused")
	ErrFileNotExist = errors.New("flisten file does not exist")
)

func asInUse(err error) error {
	if errors.Is(err, syscall.EADDRINUSE) {
		return ErrInUse
	}
	return err
}

func asFileNotExist(err error) error {
	if errors.Is(err, syscall.ENOENT) {
		return ErrFileNotExist
	}
	return err
}

func asConnRefused(err error) error {
	if errors.Is(err, syscall.ECONNREFUSED) {
		return ErrConnRefused
	}
	return err
}
