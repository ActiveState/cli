// +build linux darwin

package expect

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/kr/pty"

	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
)

func (p *Process) start() error {
	p.cmd = exec.Command(p.name, p.args...)
	p.cmd.Env = env

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

func (p *Process) wait() error {
	return p.cmd.Wait()
}

func (p *Process) close() error {
	return p.pty.Close()
}

func (p *Process) quit() error {
	return p.cmd.Process.Signal(os.Interrupt)
}

func (p *Process) exit() error {
	return p.cmd.Process.Kill()
}

func (p *Process) exitCode() int {
	return osutils.CmdExitCode(p.cmd)
}

func (p *Process) Write(input string) error {
	_, err := fmt.Fprintf(p.inWriter, "%s\n", input)
	return err
}
