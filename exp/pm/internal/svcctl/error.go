package svcctl

import (
	"context"
	"errors"

	"github.com/ActiveState/cli/exp/pm/internal/ipc"
)

var (
	errNotUp = errors.New("server not up")
)

func asNotUp(err error) error {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, ipc.ErrServerDown) {
		return errNotUp
	}
	return err
}
