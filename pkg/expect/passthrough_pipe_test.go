package expect

import (
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func prepare() (w *io.PipeWriter, p *PassthroughPipe, closer func()) {

	r, w := io.Pipe()
	p = NewPassthroughPipe(r)
	return w, p, func() {
		r.Close()
		w.Close()
		p.Close()
	}
}

func TestPassthroughPipeCustomError(t *testing.T) {
	w, p, close := prepare()
	defer close()

	p.SetReadDeadline(time.Now().Add(time.Hour))

	pipeError := errors.New("pipe error")
	err := w.CloseWithError(pipeError)
	require.NoError(t, err)

	b := make([]byte, 1)
	_, err = p.Read(b)
	require.Equal(t, err, pipeError)
}

func TestPassthroughPipeEOFError(t *testing.T) {
	w, p, close := prepare()
	defer close()

	p.SetReadDeadline(time.Now().Add(time.Hour))

	err := w.Close()
	require.NoError(t, err)

	b := make([]byte, 1)
	_, err = p.Read(b)
	require.Equal(t, err, io.EOF)
}

func TestPassthroughPipe(t *testing.T) {
	w, p, close := prepare()
	defer close()

	p.SetReadDeadline(time.Now().Add(time.Hour))

	w.Write([]byte("12abc"))
	w.Close()

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

// TestPassthroughPipeDrain drains the PassthroughPipe very slowly
// This is a regression test, ensuring that errors during reading from the pipe
// are processed *after* all bytes written to the pipe have been read
func TestPassthroughPipeDrain(t *testing.T) {
	w, p, close := prepare()
	defer close()

	p.SetReadDeadline(time.Now().Add(1 * time.Second))

	b := make([]byte, 100)
	for i := 0; i < 100; i++ {
		b[i] = byte(i)
	}
	n, err := w.Write(b)
	require.Equal(t, 100, n)
	require.NoError(t, err)
	w.Close()

	// sleep a very short while to ensure that the pipe fills up after writing to it
	time.Sleep(10 * time.Millisecond)
	b = make([]byte, 1)
	for i := 0; i < 100; i++ {
		n, err := p.Read(b)
		require.NoError(t, err)
		require.Equal(t, 1, n)
		require.Equal(t, byte(i), b[0])
	}
	n, err = p.Read(b)
	require.Error(t, err, io.EOF)
	require.Equal(t, 0, n)
}

func TestPassthroughPipeTimeout(t *testing.T) {
	w, p, close := prepare()
	defer close()

	p.SetReadDeadline(time.Now().Add(100 * time.Millisecond))

	w.Write([]byte("abc"))

	b := make([]byte, 10)
	n, err := p.Read(b)
	require.NoError(t, err)
	require.Equal(t, 3, n)
	require.Equal(t, "abc", string(b[:n]))
	n, err = p.Read(b)
	require.Error(t, err, "i/o deadline exceeded")
}
