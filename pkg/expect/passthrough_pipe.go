package expect

import (
	"context"
	"fmt"
	"io"
	"time"
)

type errPassthroughTimeout struct {
	error
}

func (errPassthroughTimeout) Timeout() bool { return true }

// bufsize is the size of the PassthroughPipe channel
const bufsize = 1024

// PassthroughPipe pipes data from a io.Reader and allows setting a read
// deadline. If a timeout is reached the error is returned, otherwise the error
// from the provided io.Reader returned is passed through instead.
type PassthroughPipe struct {
	pipeC    chan byte
	errC     chan error
	deadline time.Time
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewPassthroughPipe returns a new pipe for a io.Reader that passes through
// non-timeout errors.
func NewPassthroughPipe(r io.Reader) (p *PassthroughPipe) {
	ctx, cancel := context.WithCancel(context.Background())
	p = &PassthroughPipe{
		pipeC:    make(chan byte, bufsize),
		errC:     make(chan error, 0),
		deadline: time.Now(),
		ctx:      ctx,
		cancel:   cancel,
	}
	go func() {
		defer close(p.pipeC)
		defer close(p.errC)
		buf := make([]byte, bufsize)
	readLoop:
		for {
			n, err := r.Read(buf)

			if err != nil {
				// break on error or context timeout (note, that error channel blocks unless there is a reader (buffer size 0)
				select {
				case p.errC <- err:
				case <-ctx.Done():
				}
				break readLoop
			}
			for _, b := range buf[:n] {
				// forward the byte into the pipe channel, unless context times out
				select {
				case p.pipeC <- b:
				case <-ctx.Done():
				}
			}
		}
	}()
	return p
}

// SetReadDeadline sets a deadline for a successful read
func (p *PassthroughPipe) SetReadDeadline(d time.Time) {
	p.deadline = d
}

// Close releases all resources allocated by the pipe
func (p *PassthroughPipe) Close() error {
	p.cancel()
	return nil
}

// Flush flushes the pipe by consuming all the data written to it
func (p *PassthroughPipe) Flush() {

	buf := make([]byte, bufsize)

	for {
		n := p.consume(0, buf)
		if n == 0 {
			return
		}
	}

}

func (p *PassthroughPipe) consume(nStart int, buf []byte) int {
	ni := nStart
	for ; ni < len(buf); ni++ {
		select {
		case b := <-p.pipeC:
			buf[ni] = b
		default:
			return ni
		}
	}
	return ni
}

// Read reads from the PassthroughPipe and errors out if no data has been written to the pipe before the read deadline expired
func (p *PassthroughPipe) Read(buf []byte) (n int, err error) {

	if time.Now().After(p.deadline) {
		return 0, &errPassthroughTimeout{fmt.Errorf("i/o timeout")}
	}

	// fill buffer with bytes that are already waiting in pipe channel
	n = p.consume(0, buf)
	if n > 0 {
		return n, nil
	}

	// block until we read a byte, receive an error or time out
	select {
	case b := <-p.pipeC:
		buf[0] = b
	case e := <-p.errC:
		return 0, e
	case <-time.After(p.deadline.Sub(time.Now())):
		// force stop consuming
		p.cancel()
		return 0, &errPassthroughTimeout{fmt.Errorf("i/o timeout")}
	}

	// fill up buf or until the pipe channel is drained
	n = p.consume(1, buf)

	return n, nil
}
