package e2e

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/osutils/stacktrace"
	"github.com/ActiveState/cli/pkg/expect"
	"github.com/ActiveState/cli/pkg/expect/conproc"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/autarch/testify/require"
	"github.com/google/uuid"
	"github.com/phayes/permbits"
	"gopkg.in/yaml.v2"
)

// Session represents an end-to-end testing session during which several console process can be spawned and tested
// It provides a consistent environment (environment variables and temporary
// directories) that is shared by processes spawned during this session.
// The session is approximately the equivalent of a terminal session, with the
// main difference processes in this session are not spawned by a shell.
type Session struct {
	Dirs       *Dirs
	env        []string
	retainDirs bool
}

var (
	PersistentUsername = "cli-integration-tests"
	PersistentPassword = "test-cli-integration"

	defaultTimeout = 20 * time.Second
	authnTimeout   = 40 * time.Second
)

// executablePath returns the path to the state tool that we want to test
func (s *Session) executablePath(t *testing.T) string {
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	name := constants.CommandName + ext
	root := environment.GetRootPathUnsafe()
	subdir := "build"

	exec := filepath.Join(root, subdir, name)
	if !fileutils.FileExists(exec) {
		t.Fatal("E2E tests require a State Tool binary. Run `state run build`.")
	}

	return exec
}

func New(t *testing.T, retainDirs bool) *Session {
	dirs, err := NewDirs("")
	require.NoError(t, err)
	var env []string
	env = append(env, os.Environ()...)
	env = append(env, []string{
		"ACTIVESTATE_CLI_CONFIGDIR=" + dirs.Config,
		"ACTIVESTATE_CLI_CACHEDIR=" + dirs.Cache,
		"ACTIVESTATE_CLI_DISABLE_UPDATES=true",
		"ACTIVESTATE_CLI_DISABLE_RUNTIME=true",
		"ACTIVESTATE_PROJECT=",
	}...)

	return &Session{Dirs: dirs, env: env, retainDirs: retainDirs}
}

// Spawn spawns the state tool executable to be tested with arguments
func (s *Session) Spawn(t *testing.T, args ...string) *conproc.ConsoleProcess {
	return s.SpawnCustomWithOpts(t, s.executablePath(t), WithArgs(args...))
}

// SpawnWithOpts spawns the state tool executable to be tested with arguments
func (s *Session) SpawnWithOpts(t *testing.T, opts ...SpawnOptions) *conproc.ConsoleProcess {
	return s.SpawnCustomWithOpts(t, s.executablePath(t), opts...)
}

// SpawnCustom executes an executable in a pseudo-terminal for integration tests
func (s *Session) SpawnCustom(t *testing.T, cmdName string, args ...string) *conproc.ConsoleProcess {
	return s.SpawnCustomWithOpts(t, cmdName, WithArgs(args...))
}

// SpawnCustomWithOpts executes an executable in a pseudo-terminal for integration tests
// Arguments and other parameters can be specified by specifying SpawnOptions
func (s *Session) SpawnCustomWithOpts(t *testing.T, exe string, opts ...SpawnOptions) *conproc.ConsoleProcess {

	execu := filepath.Join(s.Dirs.Bin, filepath.Base(exe))
	fail := fileutils.CopyFile(exe, execu)
	require.NoError(t, fail.ToError())

	permissions, _ := permbits.Stat(execu)
	permissions.SetUserExecute(true)
	require.NoError(t, permbits.Chmod(execu, permissions))

	env := s.env

	pOpts := conproc.Options{
		DefaultTimeout: defaultTimeout,
		Environment:    env,
		WorkDirectory:  s.Dirs.Work,
		RetainWorkDir:  true,
		ObserveExpect:  observeExpectFn(s, t),
		ObserveSend:    observeSendFn(s, t),
		CmdName:        execu,
	}

	for _, opt := range opts {
		opt(&pOpts)
	}

	console, err := conproc.NewConsoleProcess(pOpts)
	require.NoError(t, err)

	return console
}

// PrepareActiveStateYAML creates a projectfile.Project instance from the
// provided contents and saves the output to an as.y file within the named
// directory.
func (s *Session) PrepareActiveStateYAML(t *testing.T, contents string) {
	msg := "cannot setup activestate.yaml file"

	contents = strings.TrimSpace(contents)
	projectFile := &projectfile.Project{}

	err := yaml.Unmarshal([]byte(contents), projectFile)
	require.NoError(t, err, msg)

	projectFile.SetPath(filepath.Join(s.Dirs.Work, "activestate.yaml"))
	fail := projectFile.Save()
	require.NoError(t, fail.ToError(), msg)
}

// PrepareFile writes a file to path with contents, expecting no error
func (s *Session) PrepareFile(t *testing.T, path, contents string) {
	errMsg := fmt.Sprintf("cannot setup file %q", path)

	contents = strings.TrimSpace(contents)

	err := os.MkdirAll(filepath.Dir(path), 0770)
	require.NoError(t, err, errMsg)

	bs := append([]byte(contents), '\n')

	err = ioutil.WriteFile(path, bs, 0660)
	require.NoError(t, err, errMsg)
}

// LoginAsPersistentUser is a common test case after which an integration test user should be logged in to the platform
func (s *Session) LoginAsPersistentUser(t *testing.T) {
	p := s.Spawn(t, "auth", "--username", PersistentUsername, "--password", PersistentPassword)
	defer p.Close()

	p.Expect("successfully authenticated", authnTimeout)
	p.ExpectExitCode(0)
}

func (s *Session) LogoutUser(t *testing.T) {
	p := s.Spawn(t, "auth", "logout")
	defer p.Close()

	p.Expect("logged out")
	p.ExpectExitCode(0)
}

func (s *Session) CreateNewUser(t *testing.T) string {
	uid, err := uuid.NewRandom()
	require.NoError(t, err)

	username := fmt.Sprintf("user-%s", uid.String()[0:8])
	password := username
	email := fmt.Sprintf("%s@test.tld", username)

	p := s.Spawn(t, "auth", "signup")
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

func observeSendFn(s *Session, t *testing.T) func(string, int, error) {
	return func(msg string, num int, err error) {
		if err == nil {
			return
		}

		t.Fatalf("Could not send data to terminal\nerror: %v", err)
	}
}

func observeExpectFn(s *Session, t *testing.T) func([]expect.Matcher, string, string, error) {
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

		t.Fatalf(
			"Could not meet expectation: Expectation: '%s'\nError: %v at\n%s\n---\nTerminal snapshot:\n%s\n---\nParsed output:\n%s\n",
			value, err, stacktrace.Get().String(), pty, raw,
		)
	}
}

// Close removes the temporary directory unless RetainDirs is specified
func (s *Session) Close() error {
	if s.retainDirs {
		return nil
	}
	return s.Dirs.Close()
}
