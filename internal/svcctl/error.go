package svcctl

import (
	"context"
	"errors"
	"net"
	"os"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/ipc"
)

var (
	ctlErrNotUp       = errors.New("server not up")
	ctlErrNotSureIfUp = errors.New("server may or may not be up")
)

func asNotSureIfUpErr(err error) error {
	opErr := &net.OpError{}
	if errors.Is(err, os.ErrDeadlineExceeded) || (errors.As(err, &opErr) && opErr.Timeout()) {
		return ctlErrNotSureIfUp
	}
	return err
}

func asNotUpError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) || errs.Matches(err, &ipc.ServerDownError{}) {
		return ctlErrNotUp
	}
	return err
}
