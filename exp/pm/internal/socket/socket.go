package socket

import (
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
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
	defer fmt.Println("closing listener")

	if err := os.Chmod(namespace, 0700); err != nil {
		return fmt.Errorf(emsg, err)
	}

	conns := make(chan net.Conn)

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer fmt.Println("marking pick up closed")

		for {
			if err := accept(s.done, conns, l); err != nil {
				fmt.Fprintln(os.Stderr, fmt.Errorf(emsg, err)) // wire this for return
				return
			}
		}
	}()

	for {
		if err := routeToHandler(s.done, s.wg, conns, s.mhs); err != nil {
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
			return fmt.Errorf(emsg, errors.New("done signaled"))
		default:
			return fmt.Errorf(emsg, err)
		}
	}

	conns <- conn
	return nil
}

func routeToHandler(done chan struct{}, wg *sync.WaitGroup, conns chan net.Conn, mhs []MatchedHandler) error {
	emsg := "route connection: %w"

	select {
	case <-done:
		return errors.New("done signaled")

	case conn := <-conns:
		wg.Add(1)

		go func() {
			defer wg.Done()
			defer fmt.Println("marking routing closed")
			defer conn.Close()
			defer fmt.Println("closing conn")

			if err := handleMatching(conn, mhs); err != nil {
				fmt.Fprintln(os.Stderr, fmt.Errorf(emsg, err)) // wire this for return
				return
			}
		}()

		return nil
	}
}

func handleMatching(conn net.Conn, mhs []MatchedHandler) error {
	emsg := "handle matching query: %w"

	buf := make([]byte, msgWidth)
	n, _ := conn.Read(buf) //nolint // add error and timeout handling

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
