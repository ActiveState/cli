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

	"github.com/ActiveState/termtest"
	"github.com/ActiveState/termtest/expect"
	"github.com/autarch/testify/require"
	"github.com/google/uuid"
	"github.com/phayes/permbits"
	"gopkg.in/yaml.v2"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/pkg/projectfile"
)

// Session represents an end-to-end testing session during which several console process can be spawned and tested
// It provides a consistent environment (environment variables and temporary
// directories) that is shared by processes spawned during this session.
// The session is approximately the equivalent of a terminal session, with the
// main difference processes in this session are not spawned by a shell.
type Session struct {
	cp         *termtest.ConsoleProcess
	Dirs       *Dirs
	env        []string
	retainDirs bool
	// users created during session
	users []string
	t     *testing.T
	exe   string
}

// Options for spawning a testable terminal process
type Options struct {
	termtest.Options
	// removes write-permissions in the bin directory from which executables are spawned.
	NonWriteableBinDir bool
}

var (
	PersistentUsername string
	PersistentPassword string

	defaultTimeout = 20 * time.Second
	authnTimeout   = 40 * time.Second
)

func init() {
	PersistentUsername = os.Getenv("INTEGRATION_TEST_USERNAME")
	PersistentPassword = os.Getenv("INTEGRATION_TEST_PASSWORD")
	if PersistentUsername == "" || PersistentPassword == "" {
		fmt.Println("WARNING!!! Environment variables INTEGRATION_TEST_USERNAME and INTEGRATION_TEST_PASSWORD should be defined!")
	}
}

// ExecutablePath returns the path to the state tool that we want to test
func (s *Session) ExecutablePath() string {
	return s.exe
}

// UniqueExe ensures the executable is unique to this instance
func (s *Session) UseDistinctStateExe() {
	execu := filepath.Join(s.Dirs.Bin, filepath.Base(s.exe))
	if fileutils.TargetExists(execu) {
		return
	}

	fail := fileutils.CopyFile(s.exe, execu)
	require.NoError(s.t, fail.ToError())

	// Ensure modTime is the same as source exe
	stat, err := os.Stat(s.exe)
	require.NoError(s.t, err)
	t := stat.ModTime()
	require.NoError(s.t, os.Chtimes(execu, t, t))

	permissions, _ := permbits.Stat(execu)
	permissions.SetUserExecute(true)
	require.NoError(s.t, permbits.Chmod(execu, permissions))

	s.exe = execu
}

// sourceExecutablePath returns the path to the state tool that we want to test
func executablePath(t *testing.T) string {
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

func New(t *testing.T, retainDirs bool, extraEnv ...string) *Session {
	return new(t, retainDirs, true, extraEnv...)
}

func new(t *testing.T, retainDirs, updatePath bool, extraEnv ...string) *Session {
	dirs, err := NewDirs("")
	require.NoError(t, err)
	var env []string
	env = append(env, os.Environ()...)
	env = append(env, []string{
		constants.ConfigEnvVarName + "=" + dirs.Config,
		constants.CacheEnvVarName + "=" + dirs.Cache,
		constants.DisableUpdates + "=true",
		constants.DisableRuntime + "=true",
		constants.ProjectEnvVarName + "=",
	}...)

	if updatePath {
		// add bin path
		oldPath, _ := os.LookupEnv("PATH")
		newPath := fmt.Sprintf(
			"PATH=%s%s%s",
			dirs.Bin, string(os.PathListSeparator), oldPath,
		)
		env = append(env, newPath)
	}

	// add session environment variables
	env = append(env, extraEnv...)

	return &Session{Dirs: dirs, env: env, retainDirs: retainDirs, t: t, exe: executablePath(t)}
}

func NewNoPathUpdate(t *testing.T, retainDirs bool, extraEnv ...string) *Session {
	return new(t, retainDirs, false, extraEnv...)
}

func (s *Session) ClearCache() error {
	return os.RemoveAll(s.Dirs.Cache)
}

// Spawn spawns the state tool executable to be tested with arguments
func (s *Session) Spawn(args ...string) *termtest.ConsoleProcess {
	return s.SpawnCmdWithOpts(s.exe, WithArgs(args...))
}

// SpawnWithOpts spawns the state tool executable to be tested with arguments
func (s *Session) SpawnWithOpts(opts ...SpawnOptions) *termtest.ConsoleProcess {
	return s.SpawnCmdWithOpts(s.exe, opts...)
}

// SpawnCmd executes an executable in a pseudo-terminal for integration tests
func (s *Session) SpawnCmd(cmdName string, args ...string) *termtest.ConsoleProcess {
	return s.SpawnCmdWithOpts(cmdName, WithArgs(args...))
}

// SpawnInShell runs the given command in a bash or cmd shell
func (s *Session) SpawnInShell(cmd string, opts ...SpawnOptions) *termtest.ConsoleProcess {
	exe := "/bin/bash"
	shellArgs := []string{"-c"}
	if runtime.GOOS == "windows" {
		exe = "cmd.exe"
		shellArgs = []string{"/k"}
	}

	return s.SpawnCmdWithOpts(exe, append(opts, WithArgs(append(shellArgs, cmd)...))...)
}

// SpawnCmdWithOpts executes an executable in a pseudo-terminal for integration tests
// Arguments and other parameters can be specified by specifying SpawnOptions
func (s *Session) SpawnCmdWithOpts(exe string, opts ...SpawnOptions) *termtest.ConsoleProcess {
	if s.cp != nil {
		s.cp.Close()
	}

	env := s.env

	pOpts := Options{
		Options: termtest.Options{
			DefaultTimeout: defaultTimeout,
			Environment:    env,
			WorkDirectory:  s.Dirs.Work,
			RetainWorkDir:  true,
			ObserveExpect:  observeExpectFn(s),
			ObserveSend:    observeSendFn(s),
			ExtraOpts:      []expect.ConsoleOpt{expect.WithStdout(os.Stdout)},
		},
		NonWriteableBinDir: false,
	}

	for _, opt := range opts {
		opt(&pOpts)
	}

	pOpts.Options.CmdName = exe

	if pOpts.NonWriteableBinDir {
		// make bin dir read-only
		os.Chmod(s.Dirs.Bin, 0555)
	} else {
		os.Chmod(s.Dirs.Bin, 0777)
	}

	console, err := termtest.New(pOpts.Options)
	require.NoError(s.t, err)
	s.cp = console

	return console
}

// PrepareActiveStateYAML creates a projectfile.Project instance from the
// provided contents and saves the output to an as.y file within the named
// directory.
func (s *Session) PrepareActiveStateYAML(contents string) {
	msg := "cannot setup activestate.yaml file"

	contents = strings.TrimSpace(contents)
	projectFile := &projectfile.Project{}

	err := yaml.Unmarshal([]byte(contents), projectFile)
	require.NoError(s.t, err, msg)

	projectFile.SetPath(filepath.Join(s.Dirs.Work, "activestate.yaml"))
	fail := projectFile.Save()
	require.NoError(s.t, fail.ToError(), msg)
}

// PrepareFile writes a file to path with contents, expecting no error
func (s *Session) PrepareFile(path, contents string) {
	errMsg := fmt.Sprintf("cannot setup file %q", path)

	contents = strings.TrimSpace(contents)

	err := os.MkdirAll(filepath.Dir(path), 0770)
	require.NoError(s.t, err, errMsg)

	bs := append([]byte(contents), '\n')

	err = ioutil.WriteFile(path, bs, 0660)
	require.NoError(s.t, err, errMsg)
}

func (s *Session) LoginUser(userName string) {
	p := s.Spawn("auth", "--username", userName, "--password", userName)

	p.Expect("successfully authenticated", authnTimeout)
	p.ExpectExitCode(0)
}

// LoginAsPersistentUser is a common test case after which an integration test user should be logged in to the platform
func (s *Session) LoginAsPersistentUser() {
	p := s.SpawnWithOpts(
		WithArgs("auth", "--username", PersistentUsername, "--password", PersistentPassword),
		// as the command line includes a password, we do not print the executed command, so the password does not get logged
		HideCmdLine(),
	)

	p.Expect("successfully authenticated", authnTimeout)
	p.ExpectExitCode(0)
}

func (s *Session) LogoutUser() {
	p := s.Spawn("auth", "logout")

	p.Expect("logged out")
	p.ExpectExitCode(0)
}

func (s *Session) CreateNewUser() string {
	uid, err := uuid.NewRandom()
	require.NoError(s.t, err)

	username := fmt.Sprintf("user-%s", uid.String()[0:8])
	password := username
	email := fmt.Sprintf("%s@test.tld", username)

	p := s.Spawn("auth", "signup")

	p.Expect("Terms of Service")
	p.SendLine("y")
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

	s.users = append(s.users, username)

	return username
}

func observeSendFn(s *Session) func(string, int, error) {
	return func(msg string, num int, err error) {
		if err == nil {
			return
		}

		s.t.Fatalf("Could not send data to terminal\nerror: %v", err)
	}
}

func observeExpectFn(s *Session) expect.ExpectObserver {
	return termtest.TestExpectObserveFn(s.t)
}

// Close removes the temporary directory unless RetainDirs is specified
func (s *Session) Close() error {
	if !s.retainDirs {
		defer s.Dirs.Close()
	}

	if s.cp != nil {
		s.cp.Close()
	}

	if os.Getenv("PLATFORM_API_TOKEN") == "" {
		s.t.Log("PLATFORM_API_TOKEN env var not set, not running suite tear down")
		return nil
	}

	for _, user := range s.users {
		err := cleanUser(s.t, user)
		if err != nil {
			s.t.Errorf("Could not delete user %s: %v", user, err)
		}
	}

	return nil
}

func RunningOnCI() bool {
	return os.Getenv("CI") != "" || os.Getenv("BUILDER_OUTPUT") != ""
}
