package executor

import (
	"bufio"
	"os"
	"os/exec"

	"github.com/ActiveState/cli/internal/failures"
)

var (
	FailCmdStart = failures.Type("executor.fail.cmdstart", failures.FailOS)
	FailCatch    = failures.Type("executor.fail.catch", failures.FailOS)
	FailCmdWait  = failures.Type("executor.fail.cmdwait", failures.FailOS)
	FailPipe     = failures.Type("executor.fail.pipe", failures.FailOS)
	FailScan     = failures.Type("executor.fail.scan", failures.FailOS)
)

type Executor struct {
	cmd      *exec.Cmd
	onStdin  func([]byte)
	onStdout func([]byte)
	onStderr func([]byte)
}

func New(name string, args ...string) *Executor {
	return &Executor{cmd: exec.Command(name, args...)}
}

func (e *Executor) OnStdin(cb func(input []byte)) {
	e.onStdin = cb
}

func (e *Executor) OnStdout(cb func(output []byte)) {
	e.onStdout = cb
}

func (e *Executor) OnStderr(cb func(errput []byte)) {
	e.onStderr = cb
}

func (e *Executor) Run() *failures.Failure {
	//e.cmd.Stdin = os.Stdin

	var fails []*failures.Failure
	var keepGoing = true
	defer func() { keepGoing = false }()

	// stdin
	go func() {
		err := CatchStdin(func(input []byte) bool {
			if !keepGoing {
				return false // stop
			}
			e.onStdin(input)
			return true
		})
		if err != nil {
			fails = append(fails, FailCatch.Wrap(err))
		}
	}()

	outWriter := NewStdWriter()
	outWriter.OnWrite(func(data []byte) {
		_, err := os.Stdout.Write(data)
		if err != nil {
			fails = append(fails, FailPipe.Wrap(err))
			return
		}
		e.onStdout(data)
	})
	e.cmd.Stdout = outWriter

	errWriter := NewStdWriter()
	errWriter.OnWrite(func(data []byte) {
		_, err := os.Stdout.Write(data)
		if err != nil {
			fails = append(fails, FailPipe.Wrap(err))
			return
		}
		e.onStderr(data)
	})
	e.cmd.Stderr = errWriter

	if err := e.cmd.Start(); err != nil {
		return FailCmdStart.Wrap(err)
	}

	if err := e.cmd.Wait(); err != nil {
		return FailCmdWait.Wrap(err)
	}

	if len(fails) > 0 {
		return fails[0] // can only return one, but the rest is still logged
	}

	return nil
}

func CatchStdin(callback func(input []byte) bool) error {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(bufio.ScanBytes)

	for scanner.Scan() {
		input := scanner.Bytes()
		if !callback(input) {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
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
