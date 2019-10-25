// +build darwin dragonfly linux netbsd openbsd solaris

package xpty

import (
	"io"
	"os"
	"os/exec"
	"syscall"

	"github.com/ActiveState/cli/pkg/conpty"
)

type xpty = conpty.ConPty

func Open() (*Xpty, error) {
	c, err := conpty.New(80, 20)
	if err != nil {
		return nil, err
	}
	return &Xpty{c}, nil
}

func (p *Xpty) TerminalOutPipe() io.Reader {
	return p.xpty.OutPipe()
}

func (p *Xpty) TerminalInPipe() io.Writer {
	return p.xpty.InPipe()
}

func (p *Xpty) Close() error {
	return p.xpty.Close()
}

func (p *Xpty) Tty() *os.File {
	return nil
}

func (p *Xpty) StartProcessInTerminal(c *exec.Cmd) (err error) {
	var argv []string
	if len(c.Args) > 0 {
		argv = c.Args
	} else {
		argv = []string{c.Path}
	}

	var envv []string
	if c.Env != nil {
		envv = c.Env
	} else {
		envv = os.Environ()
	}
	_, _, err = p.xpty.Spawn(c.Path, argv, &syscall.ProcAttr{
		Dir: c.Dir,
		Env: envv,
	})

	// runtime.SetFinalizer(h, )

	return
}
