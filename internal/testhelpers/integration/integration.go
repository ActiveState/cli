package integration

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/phayes/permbits"
	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v2"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/pkg/conproc"
	"github.com/ActiveState/cli/pkg/projectfile"
)

var (
	PersistentUsername = "cli-integration-tests"
	PersistentPassword = "test-cli-integration"

	defaultTimeout = 20 * time.Second
	authnTimeout   = 40 * time.Second
)

// Suite is our integration test suite
type Suite struct {
	suite.Suite
	executable string
	env        []string
	wd         *string
}

// SetupTest sets up an integration test suite for testing the State Tool executable
func (s *Suite) SetupTest() {
	exe := ""
	if runtime.GOOS == "windows" {
		exe = ".exe"
	}

	root := environment.GetRootPathUnsafe()
	executable := filepath.Join(root, "build/"+constants.CommandName+exe)

	s.wd = nil

	if !fileutils.FileExists(executable) {
		s.FailNow("Integration tests require you to have built a State Tool binary. Please run `state run build`.")
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
}

// PrepareTemporaryWorkingDirectory prepares a temporary working directory to run the tests in
// It returns the directory name a clean-up function
func (s *Suite) PrepareTemporaryWorkingDirectory(prefix string) (tempDir string, cleanup func()) {

	tempDir, err := ioutil.TempDir("", prefix)
	s.Require().NoError(err)
	err = os.RemoveAll(tempDir)
	s.Require().NoError(err)
	err = os.MkdirAll(tempDir, 0770)
	s.Require().NoError(err)
	dir, err := filepath.EvalSymlinks(tempDir)
	s.Require().NoError(err)
	s.SetWd(dir)

	return dir, func() {
		_ = os.RemoveAll(dir)
		if tempDir != dir {
			_ = os.RemoveAll(tempDir)
		}
	}
}

// PrepareActiveStateYAML creates a projectfile.Project instance from the
// provided contents and saves the output to an as.y file within the named
// directory.
func (s *Suite) PrepareActiveStateYAML(dir, contents string) {
	msg := "cannot setup activestate.yaml file"

	contents = strings.TrimSpace(contents)
	projectFile := &projectfile.Project{}

	err := yaml.Unmarshal([]byte(contents), projectFile)
	s.Require().NoError(err, msg)

	projectFile.SetPath(filepath.Join(dir, "activestate.yaml"))
	fail := projectFile.Save()
	s.Require().NoError(fail.ToError(), msg)
}

func (s *Suite) PrepareFile(path, contents string) {
	errMsg := fmt.Sprintf("cannot setup file %q", path)

	contents = strings.TrimSpace(contents)

	err := os.MkdirAll(filepath.Dir(path), 0770)
	s.Require().NoError(err, errMsg)

	bs := append([]byte(contents), '\n')

	err = ioutil.WriteFile(path, bs, 0660)
	s.Require().NoError(err, errMsg)
}

// Executable returns the path to the executable under test (State Tool)
func (s *Suite) Executable() string {
	return s.executable
}

// ClearEnv removes all environment variables
func (s *Suite) ClearEnv() {
	s.env = []string{}
}

// AppendEnv appends new environment variable settings
func (s *Suite) AppendEnv(env []string) {
	s.env = append(s.env, env...)
}

// SetWd specifies a working directory for the spawned processes.
// Use this method if you rely on running the test executable in a clean directory.
// By default all tests are run in `os.TempDir()`.
// SetWd returns a function that unsets the working directory. Use this if
// you do not want other tests to use the set directory.
func (s *Suite) SetWd(dir string) {
	s.wd = &dir
}

// Spawn executes the state tool executable under test in a pseudo-terminal
func (s *Suite) Spawn(args ...string) *ConsoleProcess {
	return s.SpawnCustom(s.executable, args...)
}

// SpawnCustom executes an executable in a pseudo-terminal for integration tests
func (s *Suite) SpawnCustom(executable string, args ...string) *conproc.ConsoleProcess {
	/*var wd string
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
	s.Require().NoError(err)

	err = console.Pty.StartProcessInTerminal(cmd)
	s.Require().NoError(err)

	ctx, cancel := context.WithCancel(context.Background())

	cp := &ConsoleProcess{
		stateCh: make(chan error),
		suite:   s.Suite,
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
	}()*/

	return nil
}

// LoginAsPersistentUser is a common test case after which an integration test user should be logged in to the platform
func (s *Suite) LoginAsPersistentUser() {
	cp := s.Spawn("auth", "--username", PersistentUsername, "--password", PersistentPassword)
	defer cp.Close()
	fmt.Println("1")
	cp.Expect("successfully authenticated", authnTimeout)
	fmt.Println("2")
	cp.ExpectExitCode(0)
	fmt.Println("3")
}

func (s *Suite) LogoutUser() {
	s.Spawn("auth", "logout")
	s.Expect("logged out")
	s.ExpectExitCode(0)
}

func (s *Suite) CreateNewUser() string {
	uid, err := uuid.NewRandom()
	s.Require().NoError(err)

	username := fmt.Sprintf("user-%s", uid.String()[0:8])
	password := username
	email := fmt.Sprintf("%s@test.tld", username)

	cp := s.Spawn("auth", "signup")
	defer cp.Close()
	cp.Expect("username:")
	cp.SendLine(username)
	cp.Expect("password:")
	cp.SendLine(password)
	cp.Expect("again:")
	cp.SendLine(password)
	cp.Expect("name:")
	cp.SendLine(username)
	cp.Expect("email:")
	cp.SendLine(email)
	cp.Expect("account has been registered", authnTimeout)
	cp.ExpectExitCode(0)

	return username
}
