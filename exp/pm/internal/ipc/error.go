package ipc

import (
	"errors"
	"syscall"
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

func asServerDown(err error) error {
	if errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.ENOENT) { // should handler per platform
		return ErrServerDown
	}
	return err
}
