package svcctl

import (
	"errors"
	"net"
	"os"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/ipc"
)

var (
	ctlErrNotUp          = errors.New("server not up")
	ctlErrTempNotUp      = errors.New("server may not be up")
	ctlErrRequestTimeout = errors.New("request timeout")
)

func asRequestTimeoutCtlErr(err error) error {
	opErr := &net.OpError{}
	if errors.Is(err, os.ErrDeadlineExceeded) || (errors.As(err, &opErr) && opErr.Timeout()) {
		return ctlErrRequestTimeout
	}
	return err
}

func asTempNotUpCtlErr(err error) error {
	if errors.Is(err, ipc.ErrConnLost) {
		return ctlErrTempNotUp
	}
	return err
}

func asNotUpCtlErr(err error) error {
	if errs.Matches(err, &ipc.ServerDownError{}) {
		return ctlErrNotUp
	}
	return err
}
