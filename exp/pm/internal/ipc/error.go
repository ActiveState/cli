package ipc

import (
	"errors"
	"fmt"

	"github.com/ActiveState/cli/exp/pm/internal/ipc/internal/flisten"
)

var (
	ErrInUse        = flisten.ErrInUse
	errConnRefused  = flisten.ErrConnRefused
	errFileNotExist = flisten.ErrFileNotExist
)

type DoneError struct {
	doneMsg string
}

func NewDoneError() *DoneError {
	return &DoneError{
		doneMsg: "done",
	}
}

func (e *DoneError) Error() string {
	return e.doneMsg
}

func (e *DoneError) DoneMsg() string {
	return e.doneMsg
}

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

func asServerDown(err error) error {
	if errors.Is(err, errFileNotExist) || errors.Is(err, errConnRefused) {
		return NewServerDownError(err)
	}
	return err
}
