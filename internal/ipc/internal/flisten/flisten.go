package flisten

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/ipc/namespace"
)

type FListen struct {
	n *namespace.Namespace
	net.Listener
}

func New(ctx context.Context, n *namespace.Namespace, network string) (*FListen, error) {
	namespace := n.String()
	namespaceDir := filepath.Dir(namespace)

	_, err := os.Stat(namespaceDir)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, errs.Wrap(err, "Cannot verify ipc dir %q", namespaceDir)
		}

		if err = os.MkdirAll(namespaceDir, 0755); err != nil {
			return nil, errs.Wrap(err, "Cannot make ipc dir %q", namespaceDir)
		}
	}

	l, err := (&net.ListenConfig{}).Listen(ctx, network, namespace)
	if err != nil {
		err = asInUse(err)
		return nil, errs.Wrap(err, "Cannot get listener for %q", namespace)
	}

	f := FListen{
		n:        n,
		Listener: l,
	}

	if err := os.Chmod(namespace, 0700); err != nil {
		_ = f.Close()
		return nil, errs.Wrap(err, "Cannot set file mode for %q", namespace)
	}

	return &f, nil
}

func NewWithCleanup(ctx context.Context, n *namespace.Namespace, network string) (*FListen, error) {
	namespace := n.String()
	if err := os.Remove(namespace); err != nil {
		return nil, errs.Wrap(err, "Cannot remove file %q", namespace)
	}

	return New(ctx, n, network)
}
