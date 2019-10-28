package integration

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"time"

	"github.com/hinshun/vt10x"
	"github.com/phayes/permbits"
	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/pkg/expect2"
)

var persistentUsername = "cli-integration-tests"
var persistentPassword = "test-cli-integration"

type Suite struct {
	suite.Suite
	console    *expect2.Console
	state      *vt10x.State
	executable string
	cmd        *exec.Cmd
	env        []string
	logFile    *os.File
}

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

	// s.ClearEnv()
	// s.AppendEnv(os.Environ())
	s.AppendEnv([]string{
		"ACTIVESTATE_CLI_CONFIGDIR=" + configDir,
		"ACTIVESTATE_CLI_CACHEDIR=" + cacheDir,
		"ACTIVESTATE_CLI_DISABLE_UPDATES=true",
		"ACTIVESTATE_CLI_DISABLE_RUNTIME=true",
		"ACTIVESTATE_PROJECT=",
		"SHELL=bash",
	})

	os.Chdir(os.TempDir())
}

func (s *Suite) ClearEnv() {
	s.env = []string{}
}

func (s *Suite) AppendEnv(env []string) {
	s.env = append(s.env, env...)
}

func (s *Suite) Spawn(args ...string) {
	wd, _ := os.Getwd()
	s.cmd = exec.Command(s.executable, args...)
	s.cmd.Dir = wd
	s.cmd.Env = s.env
	fmt.Printf("Spawning '%s' from %s\n", s.cmd.String(), wd)

	var err error
	s.logFile, err = os.Create("pty.log")
	if err != nil {
		s.Failf("", "Could not open pty log file: %v", err)
	}
	s.console, err = expect2.NewConsole(
		expect2.WithDefaultTimeout(10*time.Second),
		expect2.WithLogger(log.New(s.logFile, "", 0)),
		expect2.WithCloser(s.logFile))
	s.Require().NoError(err)

	err = s.console.Pty.StartProcessInTerminal(s.cmd)

	// stack := stacktrace.Get()
}

func (s *Suite) Output() string {
	return s.console.Pty.State.String()
}

func (s *Suite) Expect(value string, timeout ...time.Duration) {
	opts := []expect2.ExpectOpt{expect2.String(value)}
	if len(timeout) > 0 {
		opts = append(opts, expect2.WithTimeout(timeout[0]))
	}
	_, err := s.console.Expect(opts...)
	if err != nil {
		s.FailNow(
			"Could not meet expectation",
			"Expectation: '%s'\nError: %v\n---\nTerminal snapshot:\n%s\n---\n",
			value, err, s.Output())
	}
}

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

func (s *Suite) SendLine(value string) {
	_, err := s.console.SendLine(value)
	if err != nil {
		s.FailNow("Could not send data to terminal", "error: %v", err)
	}
}

func (s *Suite) Send(value string) {
	_, err := s.console.Send(value)
	if err != nil {
		s.FailNow("Could not send data to terminal", "error: %v", err)
	}
}

func (s *Suite) ExpectEOF() {
	s.console.Expect(expect2.EOF)
}

func (s *Suite) Quit() error {
	return s.cmd.Process.Signal(os.Interrupt)
}

func (s *Suite) Stop() error {
	if s.cmd == nil || s.cmd.Process == nil {
		s.FailNow("stop called without a spawned process")
	}
	return s.Quit()
}

func (s *Suite) LoginAsPersistentUser() {
	s.Spawn("auth", "--username", persistentUsername, "--password", persistentPassword)
	s.Expect("succesfully authenticated")
	s.Wait()
}

func (s *Suite) Wait() (state *os.ProcessState, err error) {
	if s.cmd == nil || s.cmd.Process == nil {
		return
	}
	return s.cmd.Process.Wait()
}
