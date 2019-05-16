package executor

import (
	"bufio"
	"io"
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
	ch := make(chan bool)

	var fails []*failures.Failure
	var keepGoing = true
	go func() {
		err := CatchStdin(e.cmd, func(input []byte) bool {
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

	stdoutPipe, err := e.cmd.StdoutPipe()
	if err != nil {
		return FailPipe.Wrap(err)
	}
	stderrPipe, err := e.cmd.StderrPipe()
	if err != nil {
		return FailPipe.Wrap(err)
	}

	if err := e.cmd.Start(); err != nil {
		return FailCmdStart.Wrap(err)
	}

	go func() {
		if err := CatchStd(stdoutPipe, os.Stdout, e.onStdout); err != nil {
			fails = append(fails, FailCatch.Wrap(err))
		}
		ch <- true
	}()
	go func() {
		if err := CatchStd(stderrPipe, os.Stderr, e.onStderr); err != nil {
			fails = append(fails, FailCatch.Wrap(err))
		}
		ch <- true
	}()

	for x := 0; x < 2; x++ {
		<-ch
	}
	keepGoing = false

	if err := e.cmd.Wait(); err != nil {
		return FailCmdWait.Wrap(err)
	}

	if len(fails) > 0 {
		return fails[0] // can only return one, but the rest is still logged
	}

	return nil
}

func CatchStdin(cmd *exec.Cmd, callback func(input []byte) bool) error {
	stdinWriter, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(bufio.ScanBytes)

	for scanner.Scan() {
		input := scanner.Bytes()
		stdinWriter.Write(input)
		if !callback(input) {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func CatchStdout(cmd *exec.Cmd, callback func(output []byte)) *failures.Failure {
	reader, err := cmd.StdoutPipe()
	if err != nil {
		return FailPipe.Wrap(err)
	}
	return CatchStd(reader, os.Stdout, callback)
}

func CatchStderr(cmd *exec.Cmd, callback func(output []byte)) *failures.Failure {
	reader, err := cmd.StderrPipe()
	if err != nil {
		return FailPipe.Wrap(err)
	}
	return CatchStd(reader, os.Stderr, callback)
}

func CatchStd(reader io.ReadCloser, std *os.File, callback func(output []byte)) *failures.Failure {
	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		output := scanner.Bytes()
		std.Write(output)
		callback(output)
	}

	if err := scanner.Err(); err != nil {
		return FailScan.Wrap(err)
	}

	return nil
}
