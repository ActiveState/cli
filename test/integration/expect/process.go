package expect

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type Process struct {
	cmd       *exec.Cmd
	onOutput  func([]byte)
	onStdout  func([]byte)
	onStderr  func([]byte)
	pty       *os.File
	stdin     io.WriteCloser
	outWriter io.Writer
	combined  string
	stdout    string
	stderr    string
	errors    []string
	running   bool
	exited    bool
}

func NewProcess(name string, args ...string) *Process {
	p := &Process{
		cmd:      exec.Command(name, args...),
		onOutput: func([]byte) {},
		onStdout: func([]byte) {},
		onStderr: func([]byte) {},
	}
	p.setupStdin()
	p.setupStdout()
	p.setupStderr()
	return p
}

func (p *Process) setupStdin() {
	var err error
	p.stdin, err = p.cmd.StdinPipe()
	if err != nil {
		panic(fmt.Sprintf("Could not pipe stdin: %v\n", err))
	}
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

func (p *Process) SetEnv(env []string) {
	p.cmd.Env = env
}

func (p *Process) OnOutput(cb func(output []byte)) {
	p.onOutput = cb
}

func (p *Process) OnStdout(cb func(output []byte)) {
	p.onStdout = cb
}

func (p *Process) OnStderr(cb func(errput []byte)) {
	p.onStderr = cb
}

func (p *Process) CombinedOutput() string { return p.combined }

func (p *Process) Stdout() string { return p.stdout }

func (p *Process) Stderr() string { return p.stderr }

func (p *Process) Running() bool { return p.running }

func (p *Process) Exited() bool { return p.exited }

func (p *Process) Quit() error {
	return p.cmd.Process.Signal(os.Interrupt)
}

func (p *Process) Exit() error {
	p.exited = true
	p.running = false
	return p.cmd.Process.Kill()
}

func (p *Process) Run() error {
	if err := p.start(); err != nil {
		return err
	}

	p.running = true
	defer func() {
		p.running = false
		p.exited = true
		p.close()
	}()

	if err := p.cmd.Wait(); err != nil {
		return err
	}

	if len(p.errors) > 0 {
		return errors.New(strings.Join(p.errors, "\n")) // can only return one, but the rest is still logged
	}

	return nil
}

type StdReader struct {
	onRead func(data []byte)
}

func NewStdReader() *StdReader {
	return &StdReader{}
}

func (w *StdReader) OnRead(cb func(data []byte)) {
	w.onRead = cb
}

func (w *StdReader) Read(p []byte) (n int, err error) {
	w.onRead(p)
	return len(p), nil
}

type StdWriter struct {
	onWrite func(data []byte)
}

func NewStdWriter() *StdWriter {
	return &StdWriter{}
}

func (w *StdWriter) OnWrite(cb func(data []byte)) {
	w.onWrite = cb
}

func (w *StdWriter) Write(p []byte) (n int, err error) {
	w.onWrite(p)
	return len(p), nil
}
