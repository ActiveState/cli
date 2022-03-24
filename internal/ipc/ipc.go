package ipc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/ActiveState/cli/internal/ipc/internal/flisten"
	"github.com/ActiveState/cli/internal/ipc/namespace"
)

var (
	msgWidth = 64
	network  = "unix"
)

type Namespace = namespace.Namespace

type MatchedHandler func(input string) (resp string, isMatched bool)

type IPC struct {
	n      *Namespace
	mhs    []MatchedHandler
	ctx    context.Context
	cancel context.CancelFunc
	wg     *sync.WaitGroup
}

func New(n *Namespace, mhs ...MatchedHandler) *IPC {
	ctx, cancel := context.WithCancel(context.Background())

	ipc := IPC{
		n:      n,
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

func (ipc *IPC) ListenAndServe() error {
	emsg := "listen and serve: %w"

	listener, err := flisten.New(ipc.ctx, ipc.n, network)
	if err != nil {
		if !errors.Is(err, flisten.ErrInUse) {
			return fmt.Errorf(emsg, err)
		}

		ctx, cancel := context.WithTimeout(ipc.ctx, time.Second*3)
		defer cancel()

		_, pingErr := NewClient(ipc.n).PingServer(ctx)
		if pingErr == nil {
			return ErrInUse
		}

		if !errors.Is(pingErr, flisten.ErrConnRefused) {
			return fmt.Errorf(emsg, pingErr) // TODO: advanced handling?
		}

		listener, err = flisten.NewWithCleanup(ctx, ipc.n, network)
		if err != nil {
			return fmt.Errorf(emsg, err)
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
					fmt.Fprintln(os.Stderr, fmt.Errorf(emsg, err)) // TODO: maybe do something more useful
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
			return err
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
	emsg := "pick up connection: %w"

	conn, err := l.Accept()
	if err != nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return fmt.Errorf(emsg, err) // TODO: should this halt the application?
		}
	}

	conns <- conn
	return nil
}

func routeToHandler(ctx context.Context, wg *sync.WaitGroup, conns chan net.Conn, mhs []MatchedHandler) error {
	emsg := "route connection: %w"

	select {
	case <-ctx.Done():
		return ctx.Err()

	case conn := <-conns:
		wg.Add(1)

		go func() {
			defer wg.Done()
			defer conn.Close()

			if err := handleMatching(conn, mhs); err != nil {
				fmt.Fprintln(os.Stderr, fmt.Errorf(emsg, err)) // TODO: maybe do something more useful
				return
			}
		}()

		return nil
	}
}

func handleMatching(conn net.Conn, mhs []MatchedHandler) error {
	emsg := "handle matching query: %w"

	buf := make([]byte, msgWidth)
	n, err := conn.Read(buf)
	if err != nil {
		return fmt.Errorf(emsg, err)
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
		return fmt.Errorf(emsg, err)
	}

	return nil
}
