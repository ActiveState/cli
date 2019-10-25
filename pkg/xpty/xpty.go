package xpty

import (
	"io"
	"os"
	"os/exec"
)

type Xpty struct {
	*xpty_ // os specific
}

func Open() (*Xpty, error) {
	xp, err := open()
	if err != nil {
		return nil, err
	}
	return &Xpty{xp}, nil
}

func (p *Xpty) TerminalOutPipe() io.Reader {
	return p.xpty_.terminalOutPipe()
}

func (p *Xpty) TerminalInPipe() io.Writer {
	return p.xpty_.terminalInPipe()
}

func (p *Xpty) Close() error {
	return p.xpty_.close()
}

func (p *Xpty) Tty() *os.File {
	return p.xpty_.tty()
}

func (p *Xpty) TerminalOutFd() uintptr {
	return p.xpty_.terminalOutFd()
}
func (p *Xpty) StartProcessInTerminal(cmd *exec.Cmd) error {
	return p.xpty_.startProcessInTerminal(cmd)
}
