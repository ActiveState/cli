package ipc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/ActiveState/cli/exp/pm/internal/ipc/internal/flisten"
	"github.com/ActiveState/cli/exp/pm/internal/ipc/namespace"
	"github.com/ActiveState/cli/exp/pm/internal/ipcerrs"
)

const (
	MsgSep = ":"
)

var (
	msgWidth = 64
	network  = "unix"
)

type Namespace = namespace.Namespace

type MatchedHandler func(input string) (resp string, isMatched bool)

type IPC struct {
	n    *Namespace
	mhs  []MatchedHandler
	done chan struct{}
	wg   *sync.WaitGroup
}

func New(n *Namespace, mhs ...MatchedHandler) *IPC {
	return &IPC{
		n:    n,
		mhs:  append(mhs, pingHandler()),
		done: make(chan struct{}),
		wg:   &sync.WaitGroup{},
	}
}

func (ipc *IPC) ListenAndServe() error {
	emsg := "listen and serve: %w"

	listener, err := flisten.New(ipc.n, network)
	if err != nil {
		if !errors.Is(err, flisten.ErrInUse) {
			return fmt.Errorf(emsg, err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()

		_, pingErr := NewClient(ipc.n).Ping(ctx)
		if pingErr == nil {
			return ErrInUse
		}

		if !errors.Is(pingErr, flisten.ErrConnRefused) {
			return fmt.Errorf(emsg, pingErr) // advanced handling?
		}

		listener, err = flisten.NewWithCleanup(ipc.n, network)
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
			if err := accept(ipc.done, conns, listener); err != nil {
				if derr := (ipcerrs.DoneError)(nil); !errors.As(err, &derr) {
					fmt.Fprintln(os.Stderr, fmt.Errorf(emsg, err)) // TODO: maybe do something more useful
				}
				return
			}
		}
	}()

	for {
		if err := routeToHandler(ipc.done, ipc.wg, conns, ipc.mhs); err != nil {
			if derr := (ipcerrs.DoneError)(nil); errors.As(err, &derr) {
				return nil
			}
			return err
		}
	}
}

func (ipc *IPC) Close() error {
	close(ipc.done)
	ipc.wg.Wait()
	return nil
}

func accept(done chan struct{}, conns chan net.Conn, l net.Listener) error {
	emsg := "pick up connection: %w"

	conn, err := l.Accept()
	if err != nil {
		select {
		case <-done:
			return fmt.Errorf(emsg, NewDoneError())
		default:
			return fmt.Errorf(emsg, err) // TODO: should this halt the application?
		}
	}

	conns <- conn
	return nil
}

func routeToHandler(done chan struct{}, wg *sync.WaitGroup, conns chan net.Conn, mhs []MatchedHandler) error {
	emsg := "route connection: %w"

	select {
	case <-done:
		return fmt.Errorf(emsg, NewDoneError())

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
