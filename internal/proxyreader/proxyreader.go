package proxyreader

import (
	"io"

	"github.com/ActiveState/cli/internal/errs"
)

type ByIncrementer interface {
	IncrBy(int)
}

var _ io.Reader = &ProxyReader{}

// ProxyReader wraps around a reader and calls the incrementer function on every read reporting the number of bytes read
type ProxyReader struct {
	increment ByIncrementer
	r         io.Reader
}

func NewProxyReader(inc ByIncrementer, r io.Reader) *ProxyReader {
	return &ProxyReader{
		increment: inc,
		r:         r,
	}
}

func (pr *ProxyReader) Read(buf []byte) (int, error) {
	n, err := pr.r.Read(buf)
	pr.increment.IncrBy(n)

	return n, err
}

// ReadAt reads into buffer starting at offset and reports progress
// Calls complete method on EOF
func (pr *ProxyReader) ReadAt(p []byte, offset int64) (int, error) {
	prAt, ok := pr.r.(io.ReaderAt)
	if !ok {
		return 0, errs.New("This proxied readers needs to implement io.ReaderAt")
	}
	n, err := prAt.ReadAt(p, offset)
	if n > 0 {
		if offset == 0 {
			pr.increment.IncrBy(n)
		}
	}
	return n, err
}
