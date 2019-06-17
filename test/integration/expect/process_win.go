// +build windows

package expect

import "io"

func (p *Process) start() error {
	return p.cmd.Start()
}

func (p *Process) close() error {
	return nil
}

func (p *Process) Write(input string) error {
	_, err := io.WriteString(p.stdin, input)
	return err
}
