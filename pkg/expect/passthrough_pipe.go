package expect

import (
	"fmt"
	"io"
	"time"
)

type errPassthroughTimeout struct {
	error
}

func (errPassthroughTimeout) Timeout() bool { return true }

// buffsize is the size of the PassthroughPipe channel
const bufsize = 1024

// PassthroughPipe pipes data from a io.Reader and allows setting a read
// deadline. If a timeout is reached the error is returned, otherwise the error
// from the provided io.Reader returned is passed through instead.
type PassthroughPipe struct {
	pipeC    chan byte
	errC     chan error
	deadline time.Time
}

// NewPassthroughPipe returns a new pipe for a io.Reader that passes through
// non-timeout errors.
func NewPassthroughPipe(r io.Reader) (p *PassthroughPipe) {
	p = &PassthroughPipe{
		pipeC:    make(chan byte, bufsize),
		errC:     make(chan error, 0),
		deadline: time.Now(),
	}
	go func() {
		defer close(p.pipeC)
		defer close(p.errC)
		buf := make([]byte, bufsize)
		for {
			n, err := r.Read(buf)
			if err != nil {
				p.errC <- err
				return
			}
			for _, b := range buf[:n] {
				p.pipeC <- b
			}
		}
	}()
	return p
}

// SetReadDeadline sets a deadline for a successful read
func (p *PassthroughPipe) SetReadDeadline(d time.Time) {
	p.deadline = d
}

// Read reads from the PassthroughPipe and errors out if no data has been written to the pipe before the read deadline expired
func (p *PassthroughPipe) Read(buf []byte) (n int, err error) {

	if time.Now().After(p.deadline) {
		return 0, &errPassthroughTimeout{fmt.Errorf("i/o timeout")}
	}

	consume := func(nStart int) int {
		ni := nStart
	fillBufLoop:
		for ; ni < len(buf); ni++ {
			select {
			case b := <-p.pipeC:
				buf[ni] = b
			default:
				break fillBufLoop
			}
		}
		return ni
	}

	// fill buffer with bytes that are already waiting in pipe channel
	n = consume(0)
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
		return 0, &errPassthroughTimeout{fmt.Errorf("i/o timeout")}
	}

	// fill up buf or until the pipe channel is drained
	n = consume(1)

	return n, nil
}
