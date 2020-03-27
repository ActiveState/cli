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
	"github.com/ActiveState/cli/pkg/expect"
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
}

func (s *Suite) executablePath() string {
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	name := constants.CommandName + ext
	root := environment.GetRootPathUnsafe()
	subdir := "build"

	exec := filepath.Join(root, subdir, name)
	if !fileutils.FileExists(exec) {
		s.FailNow("E2E tests require a State Tool binary. Run `state run build`.")
	}

	return exec
}

type SpawnOptions struct {
	Env        []string
	Dirs       *Dirs
	RetainDirs bool
}

// Spawn executes the state tool executable under test in a pseudo-terminal
func (s *Suite) Spawn(args ...string) *Process {
	return s.SpawnDirect(SpawnOptions{}, s.executablePath(), args...)
}

// SpawnCustom executes an executable in a pseudo-terminal for integration tests
func (s *Suite) SpawnCustom(opts SpawnOptions, args ...string) *Process {
	return s.SpawnDirect(opts, s.executablePath(), args...)
}

func (s *Suite) SpawnDirect(opts SpawnOptions, exe string, args ...string) *Process {
	noErr := s.Require().NoError

	if opts.Dirs == nil {
		var err error
		opts.Dirs, err = NewDirs("")
		noErr(err)
	}

	execu := filepath.Join(opts.Dirs.Bin, filepath.Base(exe))
	fail := fileutils.CopyFile(exe, execu)
	noErr(fail.ToError())

	permissions, _ := permbits.Stat(execu)
	permissions.SetUserExecute(true)
	noErr(permbits.Chmod(execu, permissions))

	var env []string
	env = append(env, os.Environ()...)
	env = append(env, []string{
		"ACTIVESTATE_CLI_CONFIGDIR=" + opts.Dirs.Config,
		"ACTIVESTATE_CLI_CACHEDIR=" + opts.Dirs.Cache,
		"ACTIVESTATE_CLI_DISABLE_UPDATES=true",
		"ACTIVESTATE_CLI_DISABLE_RUNTIME=true",
		"ACTIVESTATE_PROJECT=",
	}...)
	env = append(env, opts.Env...)

	pOpts := processOptions{
		Options: conproc.Options{
			DefaultTimeout: defaultTimeout,
			Environment:    env,
			WorkDirectory:  opts.Dirs.Work,
			RetainWorkDir:  true,
			ObserveExpect:  observeExpectFn(s),
			ObserveSend:    observeSendFn(s),
		},
		cleanUp: func() error {
			if opts.RetainDirs {
				return nil
			}
			return opts.Dirs.Close()
		},
	}

	console, err := conproc.NewConsoleProcess(pOpts.Options, execu, args...)
	noErr(err)

	return &Process{
		ConsoleProcess: console,
		pOpts:          pOpts,
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

// LoginAsPersistentUser is a common test case after which an integration test user should be logged in to the platform
func (s *Suite) LoginAsPersistentUser() {
	p := s.Spawn("auth", "--username", PersistentUsername, "--password", PersistentPassword)
	defer p.Close()

	p.Expect("successfully authenticated", authnTimeout)
	p.ExpectExitCode(0)
}

func (s *Suite) LogoutUser() {
	p := s.Spawn("auth", "logout")
	defer p.Close()

	p.Expect("logged out")
	p.ExpectExitCode(0)
}

func (s *Suite) CreateNewUser() string {
	uid, err := uuid.NewRandom()
	s.Require().NoError(err)

	username := fmt.Sprintf("user-%s", uid.String()[0:8])
	password := username
	email := fmt.Sprintf("%s@test.tld", username)

	p := s.Spawn("auth", "signup")
	defer p.Close()

	p.Expect("username:")
	p.SendLine(username)
	p.Expect("password:")
	p.SendLine(password)
	p.Expect("again:")
	p.SendLine(password)
	p.Expect("name:")
	p.SendLine(username)
	p.Expect("email:")
	p.SendLine(email)
	p.Expect("account has been registered", authnTimeout)
	p.ExpectExitCode(0)

	return username
}

func observeSendFn(s *Suite) func(string, int, error) {
	return func(msg string, num int, err error) {
		if err == nil {
			return
		}

		s.FailNow("Could not send data to terminal", "error: %v", err)
	}
}

func observeExpectFn(s *Suite) func([]expect.Matcher, string, string, error) {
	return func(matchers []expect.Matcher, raw, pty string, err error) {
		if err == nil {
			return
		}

		var value string
		var sep string
		for _, matcher := range matchers {
			value += fmt.Sprintf("%s%v", sep, matcher.Criteria())
			sep = ", "
		}

		pty = strings.TrimRight(pty, " \n") + "\n"

		s.FailNow(
			"Could not meet expectation",
			"Expectation: '%s'\nError: %v\n---\nTerminal snapshot:\n%s\n---\nParsed output:\n%s\n",
			value, err, raw, pty,
		)
	}
}
