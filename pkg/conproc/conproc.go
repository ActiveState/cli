package conproc

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/pborman/ansi"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/pkg/expect"
)

type Options struct {
	defaultTimeout time.Duration
}

type ConsoleProcess struct {
	opts    Options
	stateCh chan error
	console *expect.Console
	cmd     *exec.Cmd
	ctx     context.Context
	cancel  func()
}

// SpawnCustom executes an executable in a pseudo-terminal for integration tests
func NewConsoleProcess(executable string, args ...string) *ConsoleProcess {
	var wd string
	if s.wd == nil {
		wd = fileutils.TempDirUnsafe()
	} else {
		wd = *s.wd
	}

	cmd := exec.Command(executable, args...)
	cmd.Dir = wd
	cmd.Env = s.env

	// Create the process in a new process group.
	// This makes the behavior more consistent, as it isolates the signal handling from
	// the parent processes, which are dependent on the test environment.
	cmd.SysProcAttr = osutils.SysProcAttrForNewProcessGroup()
	fmt.Printf("Spawning '%s' from %s\n", osutils.CmdString(cmd), wd)

	var err error
	console, err := expect.NewConsole(
		expect.WithDefaultTimeout(defaultTimeout),
		expect.WithReadBufferMutation(ansi.Strip),
	)
	//s.Require().NoError(err)

	err = console.Pty.StartProcessInTerminal(cmd)
	//s.Require().NoError(err)

	ctx, cancel := context.WithCancel(context.Background())

	cp := &ConsoleProcess{
		stateCh: make(chan error),
		console: console,
		cmd:     cmd,
		ctx:     ctx,
		cancel:  cancel,
	}

	go func() {
		defer close(cp.stateCh)
		err := cmd.Wait()

		console.Close()

		fmt.Printf("send err to channel: %v\n", err)
		select {
		case cp.stateCh <- err:
		case <-cp.ctx.Done():
		}

		fmt.Printf("done sending err to channel: %v\n", err)
	}()

	return cp
}

func (cp *ConsoleProcess) Close() error {
	fmt.Println("closing channel")
	cp.cancel()
	if cp.cmd.ProcessState.Exited() {
		return nil
	}
	err := cp.cmd.Process.Kill()
	if err == nil {
		return nil
	}
	return cp.cmd.Process.Signal(syscall.SIGTERM)
}

// UnsyncedOutput returns the current Terminal snapshot.
// However the goroutine that creates this output is separate from this
// function so any output is not synced
func (cp *ConsoleProcess) UnsyncedOutput() string {
	return cp.console.Pty.State.String()
}

// ExpectRe listens to the terminal output and returns once the expected regular expression is matched or
// a timeout occurs
// Default timeout is 10 seconds
func (cp *ConsoleProcess) ExpectRe(value string, timeout ...time.Duration) {
	opts := []expect.ExpectOpt{expect.RegexpPattern(value)}
	if len(timeout) > 0 {
		opts = append(opts, expect.WithTimeout(timeout[0]))
	}
	_, err := cp.console.Expect(opts...)
	if err != nil {
		cp.suite.FailNow(
			"Could not meet expectation",
			"Expectation: '%s'\nError: %v\n---\nTerminal snapshot:\n%s\n---\n",
			value, err, cp.UnsyncedOutput())
	}
}

// TerminalSnapshot returns a snapshot of the terminal output
func (cp *ConsoleProcess) TerminalSnapshot() string {
	return cp.console.Pty.State.String()
}

// Expect listens to the terminal output and returns once the expected value is found or
// a timeout occurs
// Default timeout is 10 seconds
func (cp *ConsoleProcess) Expect(value string, timeout ...time.Duration) {
	opts := []expect.ExpectOpt{expect.String(value)}
	if len(timeout) > 0 {
		opts = append(opts, expect.WithTimeout(timeout[0]))
	}

	parsed, err := cp.console.Expect(opts...)
	if err != nil {
		cp.suite.FailNow(
			"Could not meet expectation",
			"Expectation: '%s'\nError: %v\n---\nTerminal snapshot:\n%s\n---\nParsed output:\n%s\n",
			value, err, cp.UnsyncedOutput(), parsed)
	}
}

// WaitForInput returns once a shell prompt is active on the terminal
// Default timeout is 10 seconds
func (cp *ConsoleProcess) WaitForInput(timeout ...time.Duration) {
	usr, err := user.Current()
	cp.suite.Require().NoError(err)

	msg := "echo wait_ready_$HOME"
	if runtime.GOOS == "windows" {
		msg = "echo wait_ready_%USERPROFILE%"
	}

	cp.SendLine(msg)
	cp.Expect("wait_ready_"+usr.HomeDir, timeout...)
}

// SendLine sends a new line to the terminal, as if a user typed it
func (cp *ConsoleProcess) SendLine(value string) {
	_, err := cp.console.SendLine(value)
	if err != nil {
		cp.suite.FailNow("Could not send data to terminal", "error: %v", err)
	}
}

// Send sends a string to the terminal as if a user typed it
func (cp *ConsoleProcess) Send(value string) {
	_, err := cp.console.Send(value)
	if err != nil {
		cp.suite.FailNow("Could not send data to terminal", "error: %v", err)
	}
}

// Signal sends an arbitrary signal to the running process
func (cp *ConsoleProcess) Signal(sig os.Signal) error {
	return cp.cmd.Process.Signal(sig)
}

// SendCtrlC tries to emulate what would happen in an interactive shell, when the user presses Ctrl-C
func (cp *ConsoleProcess) SendCtrlC() {
	cp.Send(string([]byte{0x03})) // 0x03 is ASCI character for ^C
}

// Quit sends an interrupt signal to the tested process
func (cp *ConsoleProcess) Quit() error {
	return cp.cmd.Process.Signal(os.Interrupt)
}

// Stop sends an interrupt signal for the tested process and fails if no process has been started yet.
func (cp *ConsoleProcess) Stop() error {
	if cp.cmd == nil || cp.cmd.Process == nil {
		cp.suite.FailNow("stop called without a spawned process")
	}
	return cp.Quit()
}

// ExpectExitCode waits for the program under test to terminate, and checks that the returned exit code meets expectations
func (cp *ConsoleProcess) ExpectExitCode(exitCode int, timeout ...time.Duration) {
	ps, err := cp.Wait(timeout...)
	if err != nil {
		cp.suite.FailNow(
			"Error waiting for process:",
			"\n%v\n---\nTerminal snapshot:\n%s\n---\n",
			err, cp.TerminalSnapshot())
	}
	if ps.ExitCode() != exitCode {
		cp.suite.FailNow(
			"Process terminated with unexpected exit code\n",
			"Expected: %d, got %d\n---\nTerminal snapshot:\n%s\n---\n",
			exitCode, ps.ExitCode(), cp.TerminalSnapshot())
	}
}

// ExpectNotExitCode waits for the program under test to terminate, and checks that the returned exit code is not the value provide
func (cp *ConsoleProcess) ExpectNotExitCode(exitCode int, timeout ...time.Duration) {
	ps, err := cp.Wait(timeout...)
	if err != nil {
		cp.suite.FailNow(
			"Error waiting for process:",
			"\n%v\n---\nTerminal snapshot:\n%s\n---\n",
			err, cp.TerminalSnapshot())
	}
	if ps.ExitCode() == exitCode {
		cp.suite.FailNow(
			"Process terminated with unexpected exit code\n",
			"Expected anything except: %d, got %d\n---\nTerminal snapshot:\n%s\n---\n",
			exitCode, ps.ExitCode(), cp.TerminalSnapshot())
	}
}

// Wait waits for the tested process to finish and returns its state including ExitCode
func (cp *ConsoleProcess) Wait(timeout ...time.Duration) (state *os.ProcessState, err error) {
	if cp.cmd == nil || cp.cmd.Process == nil {
		return
	}

	t := defaultTimeout
	if len(timeout) > 0 {
		t = timeout[0]
	}

	fmt.Printf("waiting for EOF\n")
	// TODO: This might need to be different for Windows, I think that Windows sends a different error message when we close the pseudo-terminal...
	_, err = cp.console.Expect(expect.PTSClosed, expect.EOF, expect.WithTimeout(t))
	fmt.Printf("EOF received: %v\n", err)

	if err != nil /* && err is timeout (?) */ {
		fmt.Println("killing process")
		err = cp.cmd.Process.Kill()
		if err != nil {
			// Don't know what else to do otherwise, honestly...
			panic(err)
		}
	}

	fmt.Printf("Waiting for stateCh")
	select {
	case pErr := <-cp.stateCh:
		fmt.Printf("got error: %v\n", pErr)
		return cp.cmd.ProcessState, pErr
	case <-cp.ctx.Done():
		return nil, fmt.Errorf("context canceled")
	}
}

// UnsyncedTrimSpaceOutput displays the terminal output a user would see
// however the goroutine that creates this output is separate from this
// function so any output is not synced
func (cp *ConsoleProcess) UnsyncedTrimSpaceOutput() string {
	// When the PTY reaches 80 characters it continues output on a new line.
	// On Windows this means both a carriage return and a new line. Windows
	// also picks up any spaces at the end of the console output, hence all
	// the cleaning we must do here.
	newlineRe := regexp.MustCompile(`\r?\n`)
	return newlineRe.ReplaceAllString(strings.TrimSpace(cp.UnsyncedOutput()), "")
}
