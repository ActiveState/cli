package progress

import (
	"fmt"
	"io"
	"time"

	"github.com/vbauerster/mpb/v4"
)

// ProxyReader is io.Reader wrapper, for proxy read bytes
type ProxyReader struct {
	io.ReadCloser
	bar      *mpb.Bar
	iT       time.Time
	complete func()
}

// Read reads bytes from underlying ReadCloser and reports progress
// Calls complete() method on EOF
func (pr *ProxyReader) Read(p []byte) (n int, err error) {
	n, err = pr.ReadCloser.Read(p)
	if n > 0 {
		pr.bar.IncrBy(n, time.Since(pr.iT))
		pr.iT = time.Now()
	}
	if err == io.EOF {
		go pr.complete()
	}
	return
}

// ReadAt reads into buffer starting at offset and reports progress
// Calls complete method on EOF
func (pr *ProxyReader) ReadAt(p []byte, offset int64) (n int, err error) {
	prAt, ok := pr.ReadCloser.(io.ReaderAt)
	if !ok {
		return 0, fmt.Errorf("Proxied readers needs to implement io.ReaderAt")
	}
	n, err = prAt.ReadAt(p, offset)
	if n > 0 {
		if offset == 0 || pr.bar.Current() == offset {
			pr.bar.IncrBy(n, time.Since(pr.iT))
		} else {
			pr.bar.SetCurrent(offset + int64(n))
		}
		pr.iT = time.Now()
	}
	if err == io.EOF {
		go pr.complete()
	}
	return
}
