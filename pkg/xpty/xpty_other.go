// +build darwin dragonfly linux netbsd openbsd solaris

package xpty

import (
	"io"
	"os"
	"os/exec"

	"github.com/creack/pty"
)

type xpty struct {
	ptm *os.File
	pts *os.File
}

func Open() (*Xpty, error) {
	ptm, pts, err := pty.Open()
	if err != nil {
		return nil, err
	}
	return &Xpty{&xpty{ptm, pts}}, nil
}

func (p *Xpty) TerminalOutPipe() io.Reader {
	return p.xpty.ptm
}

func (p *Xpty) TerminalInPipe() io.Writer {
	return p.xpty.ptm
}

func (p *Xpty) Close() error {
	p.xpty.pts.Close()
	p.xpty.ptm.Close()
	return nil
}

func (p *Xpty) Tty() *os.File {
	return p.xpty.pts
}

func (p *Xpty) StartProcessInTerminal(c *exec.Cmd) (err error) {
	c.StdIn = p.xpty.pts
	c.StdOut = p.xpty.pts
	c.StdErr = p.xpty.pts
	_, err = c.Start()
	return
}
