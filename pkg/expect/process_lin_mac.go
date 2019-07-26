// +build linux darwin

package expect

import (
	"fmt"
	"io"

	"github.com/kr/pty"

	"github.com/ActiveState/cli/internal/logging"
)

func (p *Process) start() error {
	var err error
	if p.pty, err = pty.Start(p.cmd); err != nil {
		return err
	}

	go func() {
		_, err := io.Copy(p.outWriter, p.pty)
		if err != nil {
			logging.Error("Error while copying stdout: %v", err)
		}
	}()

	go func() {
		_, err := io.Copy(p.pty, p.inReader)
		if err != nil {
			logging.Error("Error while copying stdin: %v", err)
		}
	}()

	return nil
}

func (p *Process) setupStdin() {
	p.inReader, p.inWriter = io.Pipe()
}

func (p *Process) setupStdout() {
	outWriter := NewStdWriter()
	outWriter.OnWrite(func(data []byte) {
		p.stdout = p.stdout + string(data)
		p.combined = p.combined + string(data)
		p.onOutput(data)
		p.onStdout(data)
	})
	p.outWriter = outWriter
}

func (p *Process) close() error {
	return p.pty.Close()
}

func (p *Process) Write(input string) error {
	_, err := fmt.Fprintf(p.inWriter, "%s\n", input)
	return err
}
