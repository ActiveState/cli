// +build darwin dragonfly linux netbsd openbsd solaris

package xpty

import (
	"io"
	"os"
	"os/exec"

	"github.com/creack/pty"
)

type xpty_ struct {
	ptm *os.File
	pts *os.File
}

func open() (*xpty_, error) {
	ptm, pts, err := pty.Open()
	if err != nil {
		return nil, err
	}
	return &xpty_{ptm, pts}, nil
}

func (p *xpty_) terminalOutPipe() io.Reader {
	return p.ptm
}

func (p *xpty_) terminalInPipe() io.Writer {
	return p.ptm
}

func (p *xpty_) close() error {
	p.pts.Close()
	p.ptm.Close()
	return nil
}

func (p *xpty_) tty() *os.File {
	return p.pts
}

func (p *xpty_) terminalOutFd() uintptr {
	return p.ptm.Fd()
}

func (p *xpty_) startProcessInTerminal(cmd *exec.Cmd) error {
	cmd.Stdin = p.pts
	cmd.Stdout = p.pts
	cmd.Stderr = p.pts
	return cmd.Start()
}
