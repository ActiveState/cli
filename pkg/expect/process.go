package expect

import (
	"io"
	"os"
	"os/exec"

	"github.com/iamacarpet/go-winpty"
)

type Process struct {
	// Command
	command string
	args    []string
	env     []string

	// Process
	cmd    *exec.Cmd
	pty    *os.File
	winpty *winpty.WinPTY

	// Event handlers
	onOutput func([]byte)
	onStdout func([]byte)
	onStderr func([]byte)

	// stdin/stdout/stderr proxies
	stdin     io.WriteCloser
	inReader  io.Reader
	inWriter  io.Writer
	outWriter io.Writer

	// Output tracking
	combined string
	stdout   string
	stderr   string

	// State
	running bool
	exited  bool
}

func NewProcess(name string, args []string, env []string) *Process {
	p := &Process{
		command:  name,
		args:     args,
		env:      env,
		onOutput: func([]byte) {},
		onStdout: func([]byte) {},
		onStderr: func([]byte) {},
	}
	p.setupStdin()
	p.setupStdout()
	p.setupStderr()
	return p
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
	p.exited = true
	p.running = false
	return p.quit()
}

func (p *Process) Exit() error {
	p.exited = true
	p.running = false
	return p.exit()
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

	if err := p.wait(); err != nil {
		return err
	}

	return nil
}

func (p *Process) ExitCode() int {
	return p.exitCode()
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
