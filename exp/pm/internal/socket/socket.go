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
	emsg := "listen: %w"
	namespace := s.n.String()

	l, err := net.Listen(network, namespace)
	if err != nil {
		return fmt.Errorf(emsg, err)
	}
	defer l.Close()

	if err = os.Chmod(namespace, 0700); err != nil {
		return fmt.Errorf(emsg, err)
	}

	conns := make(chan net.Conn)
	for {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()

			conn, err := l.Accept()
			if err != nil {
				select {
				case <-s.done:
					return
				default:
					fmt.Println(fmt.Errorf(emsg, err)) // wire this for return
					return
				}
			}
			conns <- conn
		}()

		select {
		case <-s.done:
			return errors.New("done signaled")
		case conn := <-conns:
			s.wg.Add(1)

			go func() {
				defer s.wg.Done()
				defer conn.Close()

				if err = reply(conn, s.mhs); err != nil {
					fmt.Println(fmt.Errorf(emsg, err)) // wire this for return
					return
				}
			}()
		}
	}
}

func (s *Socket) Close() error {
	close(s.done)
	s.wg.Wait()
	return nil
}

func reply(conn net.Conn, mhs []MatchedHandler) error {
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
		return err
	}

	return nil
}
