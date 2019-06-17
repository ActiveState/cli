// +build windows

package expect

import "io"

func (p *Process) start() error {
	return p.cmd.Start()
}

func (p *Process) close() error {
	return nil
}

func (p *Process) setupStdout() {
	outWriter := NewStdWriter()
	outWriter.OnWrite(func(data []byte) {
		p.stdout = p.stdout + string(data)
		p.combined = p.combined + string(data)
		p.onOutput(data)
		p.onStdout(data)
	})
	p.cmd.Stdout = outWriter
}

func (p *Process) Write(input string) error {
	_, err := io.WriteString(p.stdin, input)
	return err
}
