package ipc

import (
	"errors"
	"fmt"

	"github.com/ActiveState/cli/internal/ipc/internal/flisten"
)

var (
	// expose internal errors for outside inspection
	ErrInUse        = flisten.ErrInUse
	ErrConnRefused  = flisten.ErrConnRefused
	ErrFileNotExist = flisten.ErrFileNotExist

	ErrConnsClosed = errors.New("Connections channel closed")
)

type ServerDownError struct {
	err error
}

func NewServerDownError(err error) *ServerDownError {
	return &ServerDownError{
		err: err,
	}
}

func (e *ServerDownError) Error() string {
	return fmt.Sprintf("ipc server down: %s", e.err)
}

func (e *ServerDownError) Unwrap() error {
	return e.err
}

func asServerDownError(err error) error {
	if errors.Is(err, ErrFileNotExist) || errors.Is(err, ErrConnRefused) {
		return NewServerDownError(err)
	}
	return err
}
