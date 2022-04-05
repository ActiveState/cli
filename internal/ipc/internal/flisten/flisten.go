// Package flisten provides a type that manages a file-based unix domain socket.
// The construction funcs enable flexible usage, so abstraction of this type
// will likely require some redesign of how it is interacted with.
package flisten

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/ipc/sockpath"
)

type FListen struct {
	spath *sockpath.SockPath
	net.Listener
}

func New(ctx context.Context, spath *sockpath.SockPath, network string) (*FListen, error) {
	sockpath := spath.String()
	sockpathDir := filepath.Dir(sockpath)

	_, err := os.Stat(sockpathDir)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, errs.Wrap(err, "Cannot verify ipc dir %q", sockpathDir)
		}

		if err = os.MkdirAll(sockpathDir, 0755); err != nil {
			return nil, errs.Wrap(err, "Cannot make ipc dir %q", sockpathDir)
		}
	}

	l, err := (&net.ListenConfig{}).Listen(ctx, network, sockpath)
	if err != nil {
		err = asInUseError(err)
		return nil, errs.Wrap(err, "Cannot get listener for %q", sockpath)
	}

	f := FListen{
		spath:    spath,
		Listener: l,
	}

	if err := os.Chmod(sockpath, 0700); err != nil {
		_ = f.Close()
		return nil, errs.Wrap(err, "Cannot set file mode for %q", sockpath)
	}

	return &f, nil
}

func NewWithCleanup(ctx context.Context, spath *sockpath.SockPath, network string) (*FListen, error) {
	sockpath := spath.String()
	if err := os.Remove(sockpath); err != nil {
		return nil, errs.Wrap(err, "Cannot remove file %q", sockpath)
	}

	return New(ctx, spath, network)
}
