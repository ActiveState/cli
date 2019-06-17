// +build linux darwin

package expect

import (
	"fmt"
	"io"

	"github.com/kr/pty"
)

func (p *Process) start() error {
	var err error
	if p.pty, err = pty.Start(p.cmd); err != nil {
		return err
	}

	go func() {
		_, err := io.Copy(p.cmd.Stdout, p.pty)
		if err != nil {
			panic(fmt.Sprintf("Error while copying stdout: %v", err))
		}
		_, err = io.Copy(p.pty, p.cmd.Stdin)
		if err != nil {
			panic(fmt.Sprintf("Error while copying stdin: %v", err))
		}
	}()

	return nil
}

func (p *Process) close() error {
	return p.pty.Close()
}

func (p *Process) Write(input string) error {
	_, err := io.WriteString(p.stdin, input)
	return err
}
