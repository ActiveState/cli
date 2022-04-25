package ipc

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/ipc/internal/flisten"
	"github.com/ActiveState/cli/internal/ipc/sockpath"
	"github.com/ActiveState/cli/internal/logging"
)

var (
	msgWidth = 64
	network  = "unix"
)

type SockPath = sockpath.SockPath

// RequestHandler describes a function that receives a key which is used to
// verify if the handler is useful for a given request. If it is useful, the
// remainder of the function is used for some special behavior (usually, to
// simply return some value). This enables dynamic construction of IPC Server
// handlers/endpoints.
type RequestHandler func(key string) (resp string, isMatched bool)

type Server struct {
	spath       *SockPath
	reqHandlers []RequestHandler
	ctx         context.Context
	cancel      context.CancelFunc
	errsc       chan error
	donec       chan struct{}
}

// NewServer constructs a reference to a Server instance which can be populated
// with called-defined handlers, and is preconfigured with ping and stop
// handlers as a low-level flexibility.
func NewServer(topCtx context.Context, spath *SockPath, reqHandlers ...RequestHandler) *Server {
	ctx, cancel := context.WithCancel(topCtx)

	ipc := Server{
		spath:       spath,
		reqHandlers: make([]RequestHandler, 0, len(reqHandlers)+2),
		ctx:         ctx,
		cancel:      cancel,
		errsc:       make(chan error),
		donec:       make(chan struct{}),
	}

	ipc.reqHandlers = append(ipc.reqHandlers, pingHandler())
	ipc.reqHandlers = append(ipc.reqHandlers, reqHandlers...)
	ipc.reqHandlers = append(ipc.reqHandlers, stopHandler(ipc.Shutdown))

	return &ipc
}

func (ipc *Server) Start() error {
	listener, err := flisten.New(ipc.ctx, ipc.spath, network)
	if err != nil {
		// if sock listener construction error is "in use", ensure
		// current owner can be contacted
		if !errors.Is(err, flisten.ErrInUse) {
			return errs.Wrap(err, "Cannot construct file listener")
		}

		ctx, cancel := context.WithTimeout(ipc.ctx, time.Second*3)
		defer cancel()

		_, pingErr := NewClient(ipc.spath).PingServer(ctx)
		if pingErr == nil {
			return ErrInUse
		}

		// if client comm error is "refused", we can reasonably clobber
		// existing sock file
		if !errors.Is(pingErr, flisten.ErrConnRefused) {
			return errs.Wrap(err, "Cannot connect to existing socket file")
		}

		listener, err = flisten.NewWithCleanup(ctx, ipc.spath, network)
		if err != nil {
			return errs.Wrap(err, "Cannot construct file listener after file cleanup")
		}
	}

	// Produce (accept) and consume (route to handler) connections.
	conns := make(chan net.Conn)

	go func() {
		var wg sync.WaitGroup
		defer close(ipc.errsc)

		wg.Add(1)
		go func() {
			defer wg.Done()

			<-ipc.donec
			listener.Close()
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer close(conns)

			// Continually accept connections and feed them into the relevant channel.
			for {
				// At this time, the context.Context that is
				// passed into the flisten construction func
				// does not halt the listener. Close() must be
				// called to halt and "doneness" managed.
				err := accept(&wg, conns, listener)
				select {
				case <-ipc.donec:
					return
				default:
				}
				if err != nil {
					ipc.errsc <- errs.Wrap(err, "Unexpected accept error")
					return
				}
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()

			// Continually route incomming connections to the appropriate handler.
			for {
				if err := routeToHandler(&wg, conns, ipc.reqHandlers); err != nil {
					if !errors.Is(err, ctlErrConnsClosed) {
						logging.Error("Unexpected routeToHandler error: %v", err)
					}
					break
				}
			}
		}()

		wg.Wait()
	}()

	return nil
}

func (ipc *Server) Shutdown() {
	select {
	case <-ipc.donec:
	default:
		close(ipc.donec)
		ipc.cancel()
	}
}

func (ipc *Server) Wait() error {
	var retErr error
	for err := range ipc.errsc {
		if err != nil && retErr == nil {
			retErr = err
		}
	}
	return retErr
}

func accept(wg *sync.WaitGroup, conns chan net.Conn, l net.Listener) error {
	conn, err := l.Accept()
	if err != nil {
		return err
	}

	wg.Add(1)
	conns <- conn

	return nil
}

func routeToHandler(wg *sync.WaitGroup, conns chan net.Conn, reqHandlers []RequestHandler) error {
	select {
	case conn, ok := <-conns:
		if !ok {
			return ctlErrConnsClosed
		}

		go func() {
			defer wg.Done()
			defer conn.Close()

			if err := handleMatching(conn, reqHandlers); err != nil {
				logging.Error("unexpected ipc request handling error: %v", err)
				return
			}
		}()

		return nil
	}
}

func handleMatching(conn net.Conn, reqHandlers []RequestHandler) error {
	buf := make([]byte, msgWidth)
	n, err := conn.Read(buf)
	if err != nil {
		return errs.Wrap(err, "Failed to read from client connection")
	}

	key := string(buf[:n])
	output := "not found"

	for _, reqHandler := range reqHandlers {
		if resp, ok := reqHandler(key); ok {
			output = resp
			break
		}
	}

	if _, err := conn.Write([]byte(output)); err != nil {
		return errs.Wrap(err, "Failed to write to client connection")
	}

	return nil
}
