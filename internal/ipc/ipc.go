package ipc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/ipc/internal/flisten"
	"github.com/ActiveState/cli/internal/ipc/sockpath"
)

var (
	msgWidth = 64
	network  = "unix"
)

type SockPath = sockpath.SockPath

type MatchedHandler func(input string) (resp string, isMatched bool)

type IPC struct {
	spath  *SockPath
	mhs    []MatchedHandler
	ctx    context.Context
	cancel context.CancelFunc
	wg     *sync.WaitGroup
}

func New(spath *SockPath, mhs ...MatchedHandler) *IPC {
	ctx, cancel := context.WithCancel(context.Background())

	ipc := IPC{
		spath:  spath,
		mhs:    make([]MatchedHandler, 0, len(mhs)+2),
		ctx:    ctx,
		cancel: cancel,
		wg:     &sync.WaitGroup{},
	}

	ipc.mhs = append(ipc.mhs, pingHandler())
	ipc.mhs = append(ipc.mhs, mhs...)
	ipc.mhs = append(ipc.mhs, stopHandler(&ipc))

	return &ipc
}

func (ipc *IPC) Start() error {
	listener, err := flisten.New(ipc.ctx, ipc.spath, network)
	if err != nil {
		if !errors.Is(err, flisten.ErrInUse) {
			return errs.Wrap(err, "Cannot construct file listener")
		}

		ctx, cancel := context.WithTimeout(ipc.ctx, time.Second*3)
		defer cancel()

		_, pingErr := NewClient(ipc.spath).PingServer(ctx)
		if pingErr == nil {
			return ErrInUse
		}

		if !errors.Is(pingErr, flisten.ErrConnRefused) {
			return errs.Wrap(err, "Cannot connect to existing socket file")
		}

		listener, err = flisten.NewWithCleanup(ctx, ipc.spath, network)
		if err != nil {
			return errs.Wrap(err, "Cannot construct file listener after file cleanup")
		}
	}
	defer listener.Close()

	conns := make(chan net.Conn)

	ipc.wg.Add(1)
	go func() {
		defer ipc.wg.Done()

		for {
			if err := accept(ipc.ctx, conns, listener); err != nil {
				if !errors.Is(err, context.Canceled) {
					fmt.Fprintln(os.Stderr, fmt.Errorf("listen and server: %w", err)) // TODO: something more useful, log properly, at least
				}
				return
			}
		}
	}()

	for {
		if err := routeToHandler(ipc.ctx, ipc.wg, conns, ipc.mhs); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return errs.Wrap(err, "Critical failure handling ipc connections")
		}
	}
}

func (ipc *IPC) Close() error {
	select {
	case <-ipc.ctx.Done():
	default:
		ipc.cancel()
		ipc.wg.Wait()
	}
	return nil
}

func accept(ctx context.Context, conns chan net.Conn, l net.Listener) error {
	conn, err := l.Accept()
	if err != nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return errs.Wrap(err, "Critical error accepting connections") // TODO: should this halt the application?
		}
	}

	conns <- conn
	return nil
}

func routeToHandler(ctx context.Context, wg *sync.WaitGroup, conns chan net.Conn, mhs []MatchedHandler) error {
	select {
	case <-ctx.Done():
		return ctx.Err()

	case conn := <-conns:
		wg.Add(1)

		go func() {
			defer wg.Done()
			defer conn.Close()

			if err := handleMatching(conn, mhs); err != nil {
				fmt.Fprintln(os.Stderr, fmt.Errorf("routing connections: %w", err)) // TODO: maybe do something more useful / log properly
				return
			}
		}()

		return nil
	}
}

func handleMatching(conn net.Conn, mhs []MatchedHandler) error {
	buf := make([]byte, msgWidth)
	n, err := conn.Read(buf)
	if err != nil {
		return errs.Wrap(err, "Failed to read from connection")
	}

	input := string(buf[:n])
	output := "not found"

	for _, mh := range mhs {
		if resp, ok := mh(input); ok {
			output = resp
			break
		}
	}

	if _, err := conn.Write([]byte(output)); err != nil {
		return errs.Wrap(err, "Failed to write to connection")
	}

	return nil
}
