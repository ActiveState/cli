package progress

import (
	"fmt"
	"io"
	"time"

	"github.com/vbauerster/mpb/v4"
)

// proxyReader is io.Reader wrapper, for proxy read bytes
type proxyReader struct {
	io.ReadCloser
	bar *mpb.Bar
	iT  time.Time
}

func (pr *proxyReader) Read(p []byte) (n int, err error) {
	n, err = pr.ReadCloser.Read(p)
	if n > 0 {
		pr.bar.IncrBy(n, time.Since(pr.iT))
		pr.iT = time.Now()
	}
	if err == io.EOF {
		go func() {
			current := pr.bar.Current()
			pr.bar.SetTotal(current, true)
		}()
	}
	return
}

func (pr *proxyReader) ReadAt(p []byte, offset int64) (n int, err error) {
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
		go func() {
			current := pr.bar.Current()
			pr.bar.SetTotal(current, true)
		}()
	}
	return
}