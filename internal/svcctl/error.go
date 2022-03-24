package svcctl

import (
	"context"
	"errors"

	"github.com/ActiveState/cli/internal/ipc"
)

var (
	errNotUp = errors.New("server not up")
)

func asNotUp(err error) error {
	var sderr *ipc.ServerDownError // TODO: simplify this if possible - is it even needed?
	if errors.Is(err, context.DeadlineExceeded) || errors.As(err, &sderr) {
		return errNotUp
	}
	return err
}
