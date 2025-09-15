package output

import "io"

type WriteProxy struct {
	w io.Writer
	onWrite func(p []byte)
}

func NewWriteProxy(w io.Writer, onWrite func(p []byte)) *WriteProxy {
	return &WriteProxy{w: w, onWrite: onWrite}
}

func (w *WriteProxy) Write(p []byte) (n int, err error) {
	w.onWrite(p)
	return w.w.Write(p)
}