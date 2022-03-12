package flisten

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path"
	"syscall"

	"github.com/ActiveState/cli/exp/pm/internal/ipc/namespace"
)

var (
	ErrInUse = errors.New("flisten in use")
)

type FListen struct {
	n *namespace.Namespace
	net.Listener
}

func New(n *namespace.Namespace, network string) (*FListen, error) {
	emsg := "flisten: construct: %w"

	namespace := n.String()

	namespaceDir := path.Dir(namespace)

	_, err := os.Stat(namespaceDir)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf(emsg, err)
		}

		if err = os.MkdirAll(namespaceDir, 0755); err != nil {
			return nil, fmt.Errorf(emsg, err)
		}
	}

	l, err := net.Listen(network, namespace)
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

func asInUse(err error) error {
	if errors.Is(err, syscall.EADDRINUSE) {
		return ErrInUse
	}
	return err
}
