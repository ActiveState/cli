package expect

import (
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func prepare() (r *io.PipeReader, w *io.PipeWriter, p *PassthroughPipe, closer func()) {
	r, w = io.Pipe()
	p = NewPassthroughPipe(r)
	return r, w, p, func() {
		_ = r.Close()
		_ = w.Close()
		_ = p.Close()
	}
}

func TestPassthroughPipeCustomError(t *testing.T) {
	_, w, p, close := prepare()
	defer close()

	p.SetReadDeadline(time.Now().Add(time.Second * 2))

	pipeError := errors.New("pipe error")
	err := w.CloseWithError(pipeError)
	require.NoError(t, err)

	b := make([]byte, 1)
	n, err := p.Read(b)
	require.Equal(t, 0, n)
	require.Equal(t, pipeError, err)
}

func TestPassthroughPipeEOFError(t *testing.T) {
	_, w, p, close := prepare()
	defer close()

	p.SetReadDeadline(time.Now().Add(time.Second * 2))

	err := w.Close()
	require.NoError(t, err)

	b := make([]byte, 1)
	_, err = p.Read(b)
	require.Equal(t, io.EOF, err)
}

func TestPassthroughPipe(t *testing.T) {
	_, w, p, close := prepare()
	defer close()

	p.SetReadDeadline(time.Now().Add(time.Second * 2))

	go func() {
		_, err := w.Write([]byte("12abc"))
		require.NoError(t, err, "writing bytes")
		err = w.Close()
		require.NoError(t, err, "closing writer")
	}()

	b := make([]byte, 2)
	n, err := p.Read(b)
	require.NoError(t, err)
	require.Equal(t, 2, n)
	require.Equal(t, "12", string(b[:n]))

	b = make([]byte, 10)
	n, err = p.Read(b)
	require.NoError(t, err)
	require.Equal(t, 3, n)
	require.Equal(t, "abc", string(b[:n]))

	n, err = p.Read(b)
	require.Error(t, err, io.EOF)
	require.Equal(t, 0, n)
}

// TestPassthroughPipeReadDrain drains the PassthroughPipe very slowly
// This is a regression test, ensuring that errors during reading from the pipe
// are processed *after* all bytes written to the pipe have been read
func TestPassthroughPipeReadDrain(t *testing.T) {
	_, w, p, close := prepare()
	defer close()

	p.SetReadDeadline(time.Now().Add(1 * time.Second))

	b := make([]byte, 100)
	for i := 0; i < 100; i++ {
		b[i] = byte(i)
	}

	go func() {
		n, err := w.Write(b)
		require.Equal(t, 100, n)
		require.NoError(t, err)
		err = w.Close()
		require.NoError(t, err, "closing writer")
	}()
	// pipewriter is concurrent; sleep to let buffer fill
	time.Sleep(10 * time.Millisecond)

	b = make([]byte, 1)
	for i := 0; i < 100; i++ {
		n, err := p.Read(b)
		require.NoError(t, err)
		require.Equal(t, 1, n)
		require.Equal(t, byte(i), b[0])
	}
	n, err := p.Read(b)
	require.Error(t, err, io.EOF)
	require.Equal(t, 0, n)
}

func TestPassthroughPipeReadAfterClose(t *testing.T) {
	_, w, p, cleanup := prepare()
	defer cleanup()

	p.SetReadDeadline(time.Now().Add(200 * time.Millisecond))

	go func() {
		w.Write([]byte("abc"))
		w.Close()
	}()

	b := make([]byte, 10)
	n, err := p.Read(b)
	require.NoError(t, err)
	require.Equal(t, 3, n)
	require.Equal(t, "abc", string(b[:n]))

	n, err = p.Read(b)
	require.Error(t, err)
	require.Equal(t, 0, n)
	require.Equal(t, io.EOF, err)

	p.Close()

	n, err = p.Read(b)
	require.Error(t, err)
	require.Equal(t, 0, n)
	require.Equal(t, io.EOF, err)
}

func TestPassthroughPipeTimeout(t *testing.T) {
	_, w, p, close := prepare()
	defer close()

	p.SetReadDeadline(time.Now().Add(100 * time.Millisecond))

	go func() {
		_, err := w.Write([]byte("abc"))
		require.NoError(t, err, "writing test string")
		err = w.Close()
		require.NoError(t, err, "closing writer")
	}()

	b := make([]byte, 10)
	n, err := p.Read(b)
	require.NoError(t, err)
	require.Equal(t, 3, n)
	require.Equal(t, "abc", string(b[:n]))

	n, err = p.Read(b)
	require.Equal(t, 0, n)
	require.Error(t, err, "i/o deadline exceeded")
}

func TestPassthroughPipeClose(t *testing.T) {
	r, w, p, close := prepare()
	defer close()

	p.SetReadDeadline(time.Now().Add(100 * time.Millisecond))

	go func() {
		_, _ = w.Write([]byte("12abc"))
		_ = w.Close()
	}()

	// read one byte from passthrough
	b := make([]byte, 1)
	n, err := p.Read(b)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	require.Equal(t, "1", string(b[:n]))

	// read one byte from pipereader
	b = make([]byte, 1)
	n, err = r.Read(b)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	require.Equal(t, "2", string(b[:n]))

	err = p.Close()
	require.NoError(t, err)

	// ensure passthrough is closed
	n, err = p.Read(b)
	require.Equal(t, 0, n)
	require.Error(t, err)
	require.Equal(t, io.EOF, err)

	// ensure pipereader was drained
	b = make([]byte, 3)
	n, err = r.Read(b)
	require.Equal(t, 0, n)
	require.Equal(t, err, io.EOF)
}
