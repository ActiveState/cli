package proxyreader

import "io"

type ByIncrementer interface {
	IncrBy(int)
}

var _ io.Reader = &ProxyReader{}

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
