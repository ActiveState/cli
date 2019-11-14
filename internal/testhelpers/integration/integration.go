package integration

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"time"

	"github.com/ActiveState/vt10x"
	"github.com/phayes/permbits"
	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/pkg/expect"
)

var persistentUsername = "cli-integration-tests"
var persistentPassword = "test-cli-integration"

var defaultTimeout = 10 * time.Second

// Suite is our integration test suite
type Suite struct {
	suite.Suite
	console    *expect.Console
	state      *vt10x.State
	executable string
	cmd        *exec.Cmd
	env        []string
	wd         *string
}

// SetupTest sets up an integration test suite for testing the state tool executable
func (s *Suite) SetupTest() {
	exe := ""
	if runtime.GOOS == "windows" {
		exe = ".exe"
	}

	root := environment.GetRootPathUnsafe()
	executable := filepath.Join(root, "build/"+constants.CommandName+exe)

	if !fileutils.FileExists(executable) {
		s.FailNow("Integration tests require you to have built a state tool binary. Please run `state run build`.")
	}

	configDir, err := ioutil.TempDir("", "")
	s.Require().NoError(err)
	cacheDir, err := ioutil.TempDir("", "")
	s.Require().NoError(err)
	binDir, err := ioutil.TempDir("", "")
	s.Require().NoError(err)

	fmt.Println("Configdir: " + configDir)
	fmt.Println("Cachedir: " + cacheDir)
	fmt.Println("Bindir: " + binDir)

	s.executable = filepath.Join(binDir, constants.CommandName+exe)
	fail := fileutils.CopyFile(executable, s.executable)
	s.Require().NoError(fail.ToError())

	permissions, _ := permbits.Stat(s.executable)
	permissions.SetUserExecute(true)
	err = permbits.Chmod(s.executable, permissions)
	s.Require().NoError(err)

	s.ClearEnv()
	s.AppendEnv(os.Environ())
	s.AppendEnv([]string{
		"ACTIVESTATE_CLI_CONFIGDIR=" + configDir,
		"ACTIVESTATE_CLI_CACHEDIR=" + cacheDir,
		"ACTIVESTATE_CLI_DISABLE_UPDATES=true",
		"ACTIVESTATE_CLI_DISABLE_RUNTIME=true",
		"ACTIVESTATE_PROJECT=",
	})

	os.Chdir(os.TempDir())
}

// Executable returns the path to the executable under test (state tool)
func (s *Suite) Executable() string {
	return s.executable
}

// TeardownTest closes the terminal attached to this integration test suite
// Run this to clean-up everything set up with SetupTest()
func (s *Suite) TeardownTest() {
	s.console.Close()
}

// ClearEnv removes all environment variables
func (s *Suite) ClearEnv() {
	s.env = []string{}
}

// AppendEnv appends new environment variable settings
func (s *Suite) AppendEnv(env []string) {
	s.env = append(s.env, env...)
}

// SetWd specifies a working directory for the spawned processes
// Use this method if you rely on running the test executable in a clean directory
// By default all tests are run in `os.TempDir()`
func (s *Suite) SetWd(dir string) {
	s.wd = &dir
}

// Spawn executes the state tool executable under test in a pseudo-terminal
func (s *Suite) Spawn(args ...string) {
	s.SpawnCustom(s.executable, args...)
}

// SpawnCustom executes an executable in a pseudo-terminal for integration tests
func (s *Suite) SpawnCustom(executable string, args ...string) {
	var wd string
	if s.wd == nil {
		wd, _ = os.Getwd()
	} else {
		wd = *s.wd
	}
	s.cmd = exec.Command(executable, args...)
	s.cmd.Dir = wd
	s.cmd.Env = s.env
	fmt.Printf("Spawning '%s' from %s\n", osutils.CmdString(s.cmd), wd)

	var err error
	s.console, err = expect.NewConsole(
		expect.WithDefaultTimeout(defaultTimeout),
	)
	s.Require().NoError(err)

	err = s.console.Pty.StartProcessInTerminal(s.cmd)
	s.Require().NoError(err)
}

// Output returns the current Terminal snapshot.
func (s *Suite) Output() string {
	return s.console.Pty.State.String()
}

// ExpectRe listens to the terminal output and returns once the expected regular expression is matched or
// a timeout occurs
// Default timeout is 10 seconds
func (s *Suite) ExpectRe(value string, timeout ...time.Duration) {
	opts := []expect.ExpectOpt{expect.RegexpPattern(value)}
	if len(timeout) > 0 {
		opts = append(opts, expect.WithTimeout(timeout[0]))
	}
	_, err := s.console.Expect(opts...)
	if err != nil {
		s.FailNow(
			"Could not meet expectation",
			"Expectation: '%s'\nError: %v\n---\nTerminal snapshot:\n%s\n---\n",
			value, err, s.Output())
	}
}

// TerminalSnapshot returns a snapshot of the terminal output
func (s *Suite) TerminalSnapshot() string {
	return s.console.Pty.State.String()
}

// Expect listens to the terminal output and returns once the expected value is found or
// a timeout occurs
// Default timeout is 10 seconds
func (s *Suite) Expect(value string, timeout ...time.Duration) {
	opts := []expect.ExpectOpt{expect.String(value)}
	if len(timeout) > 0 {
		opts = append(opts, expect.WithTimeout(timeout[0]))
	}
	_, err := s.console.Expect(opts...)
	if err != nil {
		s.FailNow(
			"Could not meet expectation",
			"Expectation: '%s'\nError: %v\n---\nTerminal snapshot:\n%s\n---\n",
			value, err, s.Output())
	}
}

// WaitForInput returns once a shell prompt is active on the terminal
// Default timeout is 10 seconds
func (s *Suite) WaitForInput(timeout ...time.Duration) {
	usr, err := user.Current()
	s.Require().NoError(err)

	msg := "echo wait_ready_$HOME"
	if runtime.GOOS == "windows" {
		msg = "echo wait_ready_%USERPROFILE%"
	}

	s.SendLine(msg)
	s.Expect("wait_ready_"+usr.HomeDir, timeout...)
}

// SendLine sends a new line to the terminal, as if a user typed it
func (s *Suite) SendLine(value string) {
	_, err := s.console.SendLine(value)
	if err != nil {
		s.FailNow("Could not send data to terminal", "error: %v", err)
	}
}

// Send sends a string to the terminal as if a user typed it
func (s *Suite) Send(value string) {
	_, err := s.console.Send(value)
	if err != nil {
		s.FailNow("Could not send data to terminal", "error: %v", err)
	}
}

// ExpectEOF waits for the end of the terminal output stream before it returns
func (s *Suite) ExpectEOF() {
	s.console.Expect(expect.EOF)
}

// Quit sends an interrupt signal to the tested process
func (s *Suite) Quit() error {
	return s.cmd.Process.Signal(os.Interrupt)
}

// Stop sends an interrupt signal for the tested process and fails if no process has been started yet.
func (s *Suite) Stop() error {
	if s.cmd == nil || s.cmd.Process == nil {
		s.FailNow("stop called without a spawned process")
	}
	return s.Quit()
}

// LoginAsPersistentUser is a common test case after which an integration test user should be logged in to the platform
func (s *Suite) LoginAsPersistentUser() {
	s.Spawn("auth", "--username", persistentUsername, "--password", persistentPassword)
	s.Expect("successfully authenticated")
	s.Wait()
}

// Wait waits for the tested process to finish and returns its state including ExitCode
func (s *Suite) Wait(timeout ...time.Duration) (state *os.ProcessState, err error) {
	if s.cmd == nil || s.cmd.Process == nil {
		return
	}

	t := defaultTimeout
	if len(timeout) > 0 {
		t = timeout[0]
	}

	type processState struct {
		state *os.ProcessState
		err   error
	}
	states := make(chan processState)

	go func() {
		defer close(states)
		s, e := s.cmd.Process.Wait()
		states <- processState{state: s, err: e}
	}()

	select {
	case s := <-states:
		return s.state, s.err
	case <-time.After(t):
		return nil, fmt.Errorf("i/o error")
	}
}
