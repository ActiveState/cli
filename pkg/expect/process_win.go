// +build windows

package expect

import (
	"fmt"
	"io"
)

func (p *Process) start() error {
	return p.cmd.Start()
}

func (p *Process) close() error {
	return nil
}

func (p *Process) setupStdin() {
	var err error
	p.stdin, err = p.cmd.StdinPipe()
	if err != nil {
		panic(fmt.Sprintf("Could not pipe stdin: %v\n", err))
	}
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

func (p *Process) setupStderr() {
	errWriter := NewStdWriter()
	errWriter.OnWrite(func(data []byte) {
		p.stderr = p.stderr + string(data)
		p.combined = p.combined + string(data)
		p.onOutput(data)
		p.onStderr(data)
	})
	p.cmd.Stderr = errWriter
}
