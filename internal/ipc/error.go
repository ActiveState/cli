package ipc

import (
	"errors"
	"fmt"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/ipc/internal/flisten"
)

var (
	// expose internal errors for outside inspection
	ErrInUse = flisten.ErrInUse

	// control errors for flow control
	ctlErrConnsClosed = errors.New("Connections channel closed")
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
	if errs.IsAny(err, flisten.ErrFileNotExist, flisten.ErrConnRefused) {
		return NewServerDownError(err)
	}
	return err
}
