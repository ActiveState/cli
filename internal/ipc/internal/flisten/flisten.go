package flisten

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/ipc/namespace"
)

type FListen struct {
	n *namespace.Namespace
	net.Listener
}

func New(ctx context.Context, n *namespace.Namespace, network string) (*FListen, error) {
	emsg := "construct flisten: %w"

	namespace := n.String()

	namespaceDir := filepath.Dir(namespace)

	_, err := os.Stat(namespaceDir)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf(emsg, err)
		}

		if err = os.MkdirAll(namespaceDir, 0755); err != nil {
			return nil, fmt.Errorf(emsg, err)
		}
	}

	l, err := (&net.ListenConfig{}).Listen(ctx, network, namespace)
	if err != nil {
		err = asInUse(err)
		return nil, fmt.Errorf(emsg, err)
	}

	f := FListen{
		n:        n,
		Listener: l,
	}

	if err := os.Chmod(namespace, 0700); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf(emsg, err)
	}

	return &f, nil
}

func NewWithCleanup(ctx context.Context, n *namespace.Namespace, network string) (*FListen, error) {
	emsg := "cleanup for construction: %w"

	namespace := n.String()
	if err := os.Remove(namespace); err != nil {
		return nil, fmt.Errorf(emsg, err)
	}

	return New(ctx, n, network)
}
