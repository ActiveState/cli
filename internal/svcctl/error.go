package svcctl

import (
	"context"
	"errors"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/ipc"
)

var (
	errNotUp = errors.New("server not up")
)

func asNotUpError(err error) error {
	// TODO: simplify this if possible - is it even needed?
	if errors.Is(err, context.DeadlineExceeded) || errs.Matches(err, &ipc.ServerDownError{}) {
		return errNotUp
	}
	return err
}
