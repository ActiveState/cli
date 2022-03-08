package socket

import (
	"errors"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/ActiveState/cli/exp/pm/internal/pcerrors"
)

const (
	MsgSep = ":"
)

var (
	msgWidth = 64
	network  = "unix"
)

type MatchedHandler func(input string) (isMatched bool, resp string)

type Socket struct {
	n    *Namespace
	mhs  []MatchedHandler
	done chan struct{}
	wg   *sync.WaitGroup
}

func New(n *Namespace, mhs ...MatchedHandler) *Socket {
	return &Socket{
		n:    n,
		mhs:  mhs,
		done: make(chan struct{}),
		wg:   &sync.WaitGroup{},
	}
}

func (s *Socket) ListenAndServe() error {
	emsg := "socket: listen and serve: %w"
	namespace := s.n.String()

	l, err := net.Listen(network, namespace)
	if err != nil {
		return fmt.Errorf(emsg, err)
	}
	defer l.Close()

	if err := os.Chmod(namespace, 0700); err != nil {
		return fmt.Errorf(emsg, err)
	}

	conns := make(chan net.Conn)

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		for {
			if err := accept(s.done, conns, l); err != nil {
				if derr := (pcerrors.DoneError)(nil); !errors.As(err, &derr) {
					fmt.Fprintln(os.Stderr, fmt.Errorf(emsg, err)) // TODO: maybe do something more useful
				}
				return
			}
		}
	}()

	for {
		if err := routeToHandler(s.done, s.wg, conns, s.mhs); err != nil {
			if derr := (pcerrors.DoneError)(nil); errors.As(err, &derr) {
				return nil
			}
			return err
		}
	}
}

func (s *Socket) Close() error {
	close(s.done)
	s.wg.Wait()
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
		if ok, resp := mh(input); ok {
			output = resp
			break
		}
	}

	if _, err := conn.Write([]byte(output)); err != nil {
		return fmt.Errorf(emsg, err)
	}

	return nil
}
