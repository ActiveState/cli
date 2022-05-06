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

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/installmgr"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/rtutils/singlethread"
	"github.com/ActiveState/termtest"
	"github.com/ActiveState/termtest/expect"
	"github.com/google/uuid"
	"github.com/phayes/permbits"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
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
	users        []string
	t            *testing.T
	Exe          string
	SvcExe       string
	TrayExe      string
	InstallerExe string
}

// Options for spawning a testable terminal process
type Options struct {
	termtest.Options
	// removes write-permissions in the bin directory from which executables are spawned.
	NonWriteableBinDir bool
	// expect the process to run in background (will not be stopped by subsequent processes)
	BackgroundProcess bool
}

var (
	PersistentUsername string
	PersistentPassword string
	PersistentToken    string

	defaultTimeout = 40 * time.Second
	authnTimeout   = 40 * time.Second
)

func init() {
	PersistentUsername = os.Getenv("INTEGRATION_TEST_USERNAME")
	PersistentPassword = os.Getenv("INTEGRATION_TEST_PASSWORD")
	PersistentToken = os.Getenv("INTEGRATION_TEST_TOKEN")

	// Get username / password from `state secrets` so we can run tests without needing special env setup
	if PersistentUsername == "" {
		out, stderr, err := exeutils.ExecSimpleFromDir(environment.GetRootPathUnsafe(), "state", "secrets", "get", "project.INTEGRATION_TEST_USERNAME")
		if err != nil {
			fmt.Printf("WARNING!!! Could not retrieve username via state secrets: %v, stdout/stderr: %v\n%v\n", err, out, stderr)
		}
		PersistentUsername = strings.TrimSpace(out)
	}
	if PersistentPassword == "" {
		out, stderr, err := exeutils.ExecSimpleFromDir(environment.GetRootPathUnsafe(), "state", "secrets", "get", "project.INTEGRATION_TEST_PASSWORD")
		if err != nil {
			fmt.Printf("WARNING!!! Could not retrieve password via state secrets: %v, stdout/stderr: %v\n%v\n", err, out, stderr)
		}
		PersistentPassword = strings.TrimSpace(out)
	}
	if PersistentToken == "" {
		out, stderr, err := exeutils.ExecSimpleFromDir(environment.GetRootPathUnsafe(), "state", "secrets", "get", "project.INTEGRATION_TEST_TOKEN")
		if err != nil {
			fmt.Printf("WARNING!!! Could not retrieve token via state secrets: %v, stdout/stderr: %v\n%v\n", err, out, stderr)
		}
		PersistentToken = strings.TrimSpace(out)
	}

	if PersistentUsername == "" || PersistentPassword == "" || PersistentToken == "" {
		fmt.Println("WARNING!!! Environment variables INTEGRATION_TEST_USERNAME, INTEGRATION_TEST_PASSWORD INTEGRATION_TEST_TOKEN and should be defined!")
	}

}

// ExecutablePath returns the path to the state tool that we want to test
func (s *Session) ExecutablePath() string {
	return s.Exe
}

func (s *Session) CopyExeToDir(from, to string) string {
	to = filepath.Join(to, filepath.Base(from))
	if fileutils.TargetExists(to) {
		return to
	}

	err := fileutils.CopyFile(from, to)
	require.NoError(s.t, err, "Could not copy %s to %s", from, to)

	// Ensure modTime is the same as source exe
	stat, err := os.Stat(from)
	require.NoError(s.t, err)
	t := stat.ModTime()
	require.NoError(s.t, os.Chtimes(to, t, t))

	permissions, _ := permbits.Stat(to)
	permissions.SetUserExecute(true)
	require.NoError(s.t, permbits.Chmod(to, permissions))
	return to
}

func (s *Session) copyExeToBinDir(executable string) string {
	return s.CopyExeToDir(executable, s.Dirs.Bin)
}

// sourceExecutablePath returns the path to the state tool that we want to test
func executablePaths(t *testing.T) (string, string, string, string) {
	root := environment.GetRootPathUnsafe()
	buildDir := fileutils.Join(root, "build")

	stateExec, err := installation.NewExecInDir(buildDir, installation.StateApp)
	assert.NoError(t, err)
	svcExec, err := installation.NewExecInDir(buildDir, installation.ServiceApp)
	assert.NoError(t, err)
	trayExec, err := installation.NewExecInDir(buildDir, installation.TrayApp)
	assert.NoError(t, err)
	installExec, err := installation.NewExecInDir(buildDir, installation.InstallerApp)
	assert.NoError(t, err)

	if !fileutils.FileExists(stateExec) {
		t.Fatal("E2E tests require a State Tool binary. Run `state run build`.")
	}
	if !fileutils.FileExists(svcExec) {
		t.Fatal("E2E tests require a state-svc binary. Run `state run build-svc`.")
	}
	if !fileutils.FileExists(trayExec) {
		t.Fatal("E2E tests require a state-tray binary. Run `state run build-tray`.")
	}
	if !fileutils.FileExists(installExec) {
		t.Fatal("E2E tests require a state-installer binary. Run `state run build-installer`.")
	}

	return stateExec, svcExec, trayExec, installExec
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
		constants.DisableRuntime + "=true",
		constants.ProjectEnvVarName + "=",
		constants.E2ETestEnvVarName + "=true",
		constants.DisableUpdates + "=true",
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

	session := &Session{Dirs: dirs, env: env, retainDirs: retainDirs, t: t}

	// Mock installation directory
	exe, svcExe, trayExe, installExe := executablePaths(t)
	session.Exe = session.copyExeToBinDir(exe)
	session.SvcExe = session.copyExeToBinDir(svcExe)
	session.TrayExe = session.copyExeToBinDir(trayExe)
	session.InstallerExe = session.CopyExeToDir(installExe, dirs.Base)

	err = fileutils.Touch(filepath.Join(dirs.Base, installation.InstallDirMarker))
	require.NoError(session.t, err)

	return session
}

func NewNoPathUpdate(t *testing.T, retainDirs bool, extraEnv ...string) *Session {
	return new(t, retainDirs, false, extraEnv...)
}

func (s *Session) ClearCache() error {
	return os.RemoveAll(s.Dirs.Cache)
}

// Spawn spawns the state tool executable to be tested with arguments
func (s *Session) Spawn(args ...string) *termtest.ConsoleProcess {
	return s.SpawnCmdWithOpts(s.Exe, WithArgs(args...))
}

// SpawnWithOpts spawns the state tool executable to be tested with arguments
func (s *Session) SpawnWithOpts(opts ...SpawnOptions) *termtest.ConsoleProcess {
	return s.SpawnCmdWithOpts(s.Exe, opts...)
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
	if !pOpts.BackgroundProcess {
		s.cp = console
	}

	logging.Debug("Spawning CMD: %s, args: %v", pOpts.Options.CmdName, pOpts.Options.Args)

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

	cfg, err := config.New()
	require.NoError(s.t, err)
	defer func() { require.NoError(s.t, cfg.Close()) }()

	projectFile.SetPath(filepath.Join(s.Dirs.Work, "activestate.yaml"))
	err = projectFile.Save(cfg)
	require.NoError(s.t, err, msg)
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
	p := s.Spawn(tagsuite.Auth, "--username", userName, "--password", userName)

	p.Expect("logged in", authnTimeout)
	p.ExpectExitCode(0)
}

// LoginAsPersistentUser is a common test case after which an integration test user should be logged in to the platform
func (s *Session) LoginAsPersistentUser() {
	p := s.SpawnWithOpts(
		WithArgs(tagsuite.Auth, "--username", PersistentUsername, "--password", PersistentPassword),
		// as the command line includes a password, we do not print the executed command, so the password does not get logged
		HideCmdLine(),
	)

	p.Expect("logged in", authnTimeout)
	p.ExpectExitCode(0)
}

func (s *Session) LogoutUser() {
	p := s.Spawn(tagsuite.Auth, "logout")

	p.Expect("logged out")
	p.ExpectExitCode(0)
}

func (s *Session) CreateNewUser() string {
	uid, err := uuid.NewRandom()
	require.NoError(s.t, err)

	username := fmt.Sprintf("user-%s", uid.String()[0:8])
	password := username
	email := fmt.Sprintf("%s@test.tld", username)

	p := s.Spawn(tagsuite.Auth, "signup", "--prompt")

	p.Expect("Terms of Service")
	p.Send("y")
	p.Expect("username:")
	p.Send(username)
	p.Expect("password:")
	p.Send(password)
	p.Expect("again:")
	p.Send(password)
	p.Expect("email:")
	p.Send(email)
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
	// stop service and tray if they exist
	if fileutils.TargetExists(s.SvcExe) {
		cp := s.SpawnCmd(s.SvcExe, "stop")
		cp.ExpectExitCode(0)
	}

	cfg, err := config.NewCustom(s.Dirs.Config, singlethread.New(), true)
	require.NoError(s.t, err, "Could not read e2e session configuration: %s", errs.JoinMessage(err))
	err = installmgr.StopTrayApp(cfg)
	require.NoError(s.t, err, "Could not stop tray app")

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
	return condition.OnCI()
}
