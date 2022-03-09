package ipc

import (
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"syscall"

	"github.com/ActiveState/cli/exp/pm/internal/ipc/internal/flisten"
	"github.com/ActiveState/cli/exp/pm/internal/ipc/namespace"
	"github.com/ActiveState/cli/exp/pm/internal/pcerrors"
)

const (
	MsgSep = ":"
)

var (
	msgWidth = 64
	network  = "unix"

	ErrInUse       = flisten.ErrInUse
	ErrConnRefused = errors.New("ipc connection refused")
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
		mhs:  append(mhs, internalPingHandler()),
		done: make(chan struct{}),
		wg:   &sync.WaitGroup{},
	}
}

func (c *IPC) ListenAndServe() error {
	emsg := "socket: listen and serve: %w"

	l, err := flisten.New(c.n, network)
	if err != nil {
		if errors.Is(err, flisten.ErrInUse) {
			_, pingErr := getPing(NewClient(c.n))
			if pingErr != nil {
				if errors.Is(pingErr, syscall.ECONNREFUSED) {
					return fmt.Errorf(emsg, ErrConnRefused) // should take down sock file and retry
				}

				return fmt.Errorf(emsg, pingErr) // advanced handling?
			}
			return ErrInUse // should die - check for in main specifically
		}

		return fmt.Errorf(emsg, err)
	}
	defer l.Close()

	conns := make(chan net.Conn)

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()

		for {
			if err := accept(c.done, conns, l); err != nil {
				if derr := (pcerrors.DoneError)(nil); !errors.As(err, &derr) {
					fmt.Fprintln(os.Stderr, fmt.Errorf(emsg, err)) // TODO: maybe do something more useful
				}
				return
			}
		}
	}()

	for {
		if err := routeToHandler(c.done, c.wg, conns, c.mhs); err != nil {
			if derr := (pcerrors.DoneError)(nil); errors.As(err, &derr) {
				return nil
			}
			return err
		}
	}
}

func (c *IPC) Close() error {
	close(c.done)
	c.wg.Wait()
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
