package flisten

import (
	"errors"
	"syscall"
)

var (
	ErrInUse        = errors.New("flisten in use")
	ErrConnRefused  = errors.New("flisten connection refused")
	ErrConnLost     = errors.New("flisten connection lost")
	ErrFileNotExist = errors.New("flisten file does not exist")
)

func asFileNotExistError(err error) error {
	if errors.Is(err, syscall.ENOENT) {
		return ErrFileNotExist
	}
	return err
}
