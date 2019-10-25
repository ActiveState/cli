package integration

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

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
	executable string
	cmd        *exec.Cmd
	env        []string
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
	s.console, err = expect2.NewConsole()
	s.Require().NoError(err)

	err = s.console.Pty.StartProcessInTerminal(s.cmd)

	// stack := stacktrace.Get()
}

func (s *Suite) Expect(value string) {
	_, err := s.console.ExpectString(value)
	s.Require().NoError(err)
}

func (s *Suite) Wait() {
	s.console.ExpectEOF()
}

func (s *Suite) LoginAsPersistentUser() {
	s.Spawn("auth", "--username", persistentUsername, "--password", persistentPassword)
	s.Expect("succesfully authenticated")
	s.Wait()
}
