// +build windows

package xpty

import (
	"io"
	"os"
	"os/exec"
	"syscall"

	"github.com/ActiveState/cli/pkg/conpty"
)

type xpty_ struct {
	*conpty.ConPty
}

func open() (*xpty_, error) {
	c, err := conpty.New(80, 20)
	if err != nil {
		return nil, err
	}
	return &xpty_{c}, nil
}

func (p *xpty_) terminalOutPipe() io.Reader {
	return p.OutPipe()
}

func (p *xpty_) terminalInPipe() io.Writer {
	return p.InPipe()
}

func (p *xpty_) close() error {
	return p.Close()
}

func (p *xpty_) tty() *os.File {
	return nil
}

func (p *xpty_) terminalOutFd() uintptr {
	return p.OutFd()
}

func (p *xpty_) startProcessInTerminal(c *exec.Cmd) (err error) {
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
	_, _, err = p.Spawn(c.Path, argv, &syscall.ProcAttr{
		Dir: c.Dir,
		Env: envv,
	})

	// runtime.SetFinalizer(h, )

	return
}
