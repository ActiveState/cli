package e2e

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/osutils/stacktrace"
	"github.com/ActiveState/cli/internal/rtutils/singlethread"
	"github.com/ActiveState/cli/internal/strutils"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/subshell/bash"
	"github.com/ActiveState/cli/internal/subshell/sscommon"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/model/buildplanner"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/ActiveState/termtest"
	"github.com/go-openapi/strfmt"
	"github.com/phayes/permbits"
	"github.com/stretchr/testify/require"
)

var (
	RuntimeSolvingTimeoutOpt       = termtest.OptExpectTimeout(90 * time.Second)
	RuntimeSourcingTimeoutOpt      = termtest.OptExpectTimeout(3 * time.Minute)
	RuntimeBuildSourcingTimeoutOpt = termtest.OptExpectTimeout(RuntimeBuildSourcingTimeout)
)

// Session represents an end-to-end testing session during which several console process can be spawned and tested
// It provides a consistent environment (environment variables and temporary
// directories) that is shared by processes spawned during this session.
// The session is approximately the equivalent of a terminal session, with the
// main difference processes in this session are not spawned by a shell.
type Session struct {
	Dirs            *Dirs
	Env             []string
	retainDirs      bool
	createdProjects []*project.Namespaced
	// users created during session
	users           []string
	T               *testing.T
	Exe             string
	SvcExe          string
	ExecutorExe     string
	spawned         []*SpawnedCmd
	ignoreLogErrors bool
	cache           keyCache
}

type keyCache map[string]string

func (k keyCache) GetCache(key string) (string, error) {
	return k[key], nil
}

func (k keyCache) SetCache(key, value string, _ time.Duration) error {
	k[key] = value
	return nil
}

var (
	PersistentUsername string
	PersistentPassword string
	PersistentToken    string

	defaultTimeout         = 40 * time.Second
	RuntimeSourcingTimeout = 3 * time.Minute
)

func init() {
	PersistentUsername = os.Getenv("INTEGRATION_TEST_USERNAME")
	PersistentPassword = os.Getenv("INTEGRATION_TEST_PASSWORD")
	PersistentToken = os.Getenv("INTEGRATION_TEST_TOKEN")

	// Get username / password from `state secrets` so we can run tests without needing special env setup
	if PersistentUsername == "" {
		out, stderr, err := osutils.ExecSimpleFromDir(environment.GetRootPathUnsafe(), "state", []string{"secrets", "get", "project.INTEGRATION_TEST_USERNAME"}, []string{})
		if err != nil {
			fmt.Printf("WARNING!!! Could not retrieve username via state secrets: %v, stdout/stderr: %v\n%v\n", err, out, stderr)
		}
		PersistentUsername = strings.TrimSpace(out)
	}
	if PersistentPassword == "" {
		out, stderr, err := osutils.ExecSimpleFromDir(environment.GetRootPathUnsafe(), "state", []string{"secrets", "get", "project.INTEGRATION_TEST_PASSWORD"}, []string{})
		if err != nil {
			fmt.Printf("WARNING!!! Could not retrieve password via state secrets: %v, stdout/stderr: %v\n%v\n", err, out, stderr)
		}
		PersistentPassword = strings.TrimSpace(out)
	}
	if PersistentToken == "" {
		out, stderr, err := osutils.ExecSimpleFromDir(environment.GetRootPathUnsafe(), "state", []string{"secrets", "get", "project.INTEGRATION_TEST_TOKEN"}, []string{})
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
	var err error
	to, err = filepath.Abs(filepath.Join(to, filepath.Base(from)))
	if err != nil {
		s.T.Fatal(err)
	}
	if fileutils.TargetExists(to) {
		return to
	}

	err = fileutils.CopyFile(from, to)
	require.NoError(s.T, err, "Could not copy %s to %s", from, to)

	// Ensure modTime is the same as source exe
	stat, err := os.Stat(from)
	require.NoError(s.T, err)
	t := stat.ModTime()
	require.NoError(s.T, os.Chtimes(to, t, t))

	permissions, _ := permbits.Stat(to)
	permissions.SetUserExecute(true)
	require.NoError(s.T, permbits.Chmod(to, permissions))
	return to
}

func (s *Session) copyExeToBinDir(executable string) string {
	return s.CopyExeToDir(executable, s.Dirs.Bin)
}

// executablePaths returns the paths to the executables that we want to test
func executablePaths(t *testing.T) (string, string, string) {
	root := environment.GetRootPathUnsafe()
	buildDir := fileutils.Join(root, "build")

	stateExec := filepath.Join(buildDir, constants.StateCmd+osutils.ExeExtension)
	svcExec := filepath.Join(buildDir, constants.StateSvcCmd+osutils.ExeExtension)
	executorExec := filepath.Join(buildDir, constants.StateExecutorCmd+osutils.ExeExtension)

	if !fileutils.FileExists(stateExec) {
		t.Fatal("E2E tests require a State Tool binary. Run `state run build`.")
	}
	if !fileutils.FileExists(svcExec) {
		t.Fatal("E2E tests require a state-svc binary. Run `state run build-svc`.")
	}
	if !fileutils.FileExists(executorExec) {
		t.Fatal("E2E tests require a state-exec binary. Run `state run build-exec`.")
	}

	return stateExec, svcExec, executorExec
}

func New(t *testing.T, retainDirs bool, extraEnv ...string) *Session {
	return new(t, retainDirs, true, extraEnv...)
}

func new(t *testing.T, retainDirs, updatePath bool, extraEnv ...string) *Session {
	dirs, err := NewDirs("")
	require.NoError(t, err)
	env := sandboxedTestEnvironment(t, dirs, updatePath, extraEnv...)

	session := &Session{Dirs: dirs, Env: env, retainDirs: retainDirs, T: t, cache: keyCache{}}

	// Mock installation directory
	exe, svcExe, execExe := executablePaths(t)
	session.Exe = session.copyExeToBinDir(exe)
	session.SvcExe = session.copyExeToBinDir(svcExe)
	session.ExecutorExe = session.copyExeToBinDir(execExe)

	err = fileutils.Touch(filepath.Join(dirs.Base, installation.InstallDirMarker))
	require.NoError(session.T, err)

	cfg, err := config.NewCustom(dirs.Config, singlethread.New(), true)
	require.NoError(session.T, err)

	if err := cfg.Set(constants.SecurityPromptConfig, false); err != nil {
		require.NoError(session.T, err)
	}

	return session
}

func NewNoPathUpdate(t *testing.T, retainDirs bool, extraEnv ...string) *Session {
	return new(t, retainDirs, false, extraEnv...)
}

func (s *Session) SetT(t *testing.T) {
	s.T = t
}

func (s *Session) ClearCache() error {
	return os.RemoveAll(s.Dirs.Cache)
}

// Spawn spawns the state tool executable to be tested with arguments
func (s *Session) Spawn(args ...string) *SpawnedCmd {
	return s.SpawnCmdWithOpts(s.Exe, OptArgs(args...))
}

// SpawnDebuggerWithOpts will spawn a state tool command with the dlv debugger in remote debugging port.
// It uses the default dlv port of `2345`. It has been tested in Goland (intellij), see instructions here:
// https://www.jetbrains.com/help/go/attach-to-running-go-processes-with-debugger.html#step-3-create-the-remote-run-debug-configuration-on-the-client-computer
// Note remote debugging seems a bit unreliable. I've found it works best to start the test code first, and once it is
// running then start the remote debugger. When I launch the remote debugger first it often doesn't take. But even
// when using this trickery it may at times not work; try restarting goland, your machine, or dlv.
func (s *Session) SpawnDebuggerWithOpts(opts ...SpawnOptSetter) *SpawnedCmd {
	spawnOpts := s.newSpawnOpts(opts...)
	args := slices.Clone(spawnOpts.Args)

	workDir := spawnOpts.Dir
	spawnOpts.Args = []string{"debug", "--wd", workDir, "--headless", "--listen=:2345", "--api-version=2", "github.com/ActiveState/cli/cmd/state", "--"}
	spawnOpts.Args = append(spawnOpts.Args, args...)
	spawnOpts.Dir = environment.GetRootPathUnsafe()

	return s.SpawnCmdWithOpts("dlv", func(opts *SpawnOpts) {
		*opts = spawnOpts
	})
}

// SpawnWithOpts spawns the state tool executable to be tested with arguments
func (s *Session) SpawnWithOpts(opts ...SpawnOptSetter) *SpawnedCmd {
	return s.SpawnCmdWithOpts(s.Exe, opts...)
}

// SpawnCmd executes an executable in a pseudo-terminal for integration tests
func (s *Session) SpawnCmd(cmdName string, args ...string) *SpawnedCmd {
	return s.SpawnCmdWithOpts(cmdName, OptArgs(args...))
}

// SpawnShellWithOpts spawns the given shell and options in interactive mode.
func (s *Session) SpawnShellWithOpts(shell Shell, opts ...SpawnOptSetter) *SpawnedCmd {
	if shell != Cmd {
		opts = append(opts, OptAppendEnv("SHELL="+string(shell)))
	}
	opts = append(opts, OptRunInsideShell(false))
	return s.SpawnCmdWithOpts(string(shell), opts...)
}

// SpawnCmdWithOpts executes an executable in a pseudo-terminal for integration tests
// Arguments and other parameters can be specified by specifying SpawnOptSetter
func (s *Session) SpawnCmdWithOpts(exe string, optSetters ...SpawnOptSetter) *SpawnedCmd {
	spawnOpts := s.newSpawnOpts(optSetters...)

	var shell string
	var args []string
	if spawnOpts.RunInsideShell {
		switch runtime.GOOS {
		case "windows":
			shell = Cmd
			// /C = next argument is command that will be ran
			args = []string{"/C"}
		case "darwin":
			shell = "zsh"
			// -i = interactive mode
			// -c = next argument is command that will be ran
			args = []string{"-i", "-c"}
		default:
			shell = "bash"
			args = []string{"-i", "-c"}
		}
		if len(spawnOpts.Args) == 0 {
			args = append(args, fmt.Sprintf(`"%s"`, exe))
		} else {
			if shell == Cmd {
				aa := spawnOpts.Args
				for i, a := range aa {
					aa[i] = strings.ReplaceAll(a, " ", "^ ")
				}
				// Windows is weird, it doesn't seem to let you quote arguments, so instead we need to escape spaces
				// which on Windows is done with the '^' character.
				args = append(args, fmt.Sprintf(`%s %s`, strings.ReplaceAll(exe, " ", "^ "), strings.Join(aa, " ")))
			} else {
				args = append(args, fmt.Sprintf(`"%s" "%s"`, exe, strings.Join(spawnOpts.Args, `" "`)))
			}
		}
	} else {
		shell = exe
		args = spawnOpts.Args
	}

	cmd := osutils.Command(shell, args...)

	cmd.Env = spawnOpts.Env
	if spawnOpts.Dir != "" {
		cmd.Dir = spawnOpts.Dir
	}

	tt, err := termtest.New(cmd, spawnOpts.TermtestOpts...)
	require.NoError(s.T, err)

	spawn := &SpawnedCmd{tt, spawnOpts}

	s.spawned = append(s.spawned, spawn)

	cmdArgs := spawnOpts.Args
	if spawnOpts.HideCmdArgs {
		cmdArgs = []string{"<hidden>"}
	}
	logging.Debug("Spawning CMD: %s, args: %v", exe, cmdArgs)

	return spawn
}

func (s *Session) newSpawnOpts(optSetters ...SpawnOptSetter) SpawnOpts {
	spawnOpts := NewSpawnOpts()
	spawnOpts.Env = s.Env
	spawnOpts.Dir = s.Dirs.Work

	spawnOpts.TermtestOpts = append(spawnOpts.TermtestOpts,
		termtest.OptErrorHandler(func(tt *termtest.TermTest, err error) error {
			s.T.Fatal(s.DebugMessage(errs.JoinMessage(err)))
			return err
		}),
		termtest.OptDefaultTimeout(defaultTimeout),
		termtest.OptCols(140),
		termtest.OptRows(30), // Needs to be able to accommodate most JSON output
	)

	// TTYs output newlines in two steps: '\r' (CR) to move the caret to the beginning of the line,
	// and '\n' (LF) to move the caret one line down. Terminal emulators do the same thing, so the
	// raw terminal output will contain "\r\n". Since our multi-line expectation messages often use
	// '\n', normalize line endings to that for convenience, regardless of platform ('\n' for Linux
	// and macOS, "\r\n" for Windows).
	// More info: https://superuser.com/a/1774370
	spawnOpts.TermtestOpts = append(spawnOpts.TermtestOpts,
		termtest.OptNormalizedLineEnds(true),
	)

	for _, optSet := range optSetters {
		optSet(&spawnOpts)
	}

	return spawnOpts
}

// PrepareActiveStateYAML creates an activestate.yaml in the session's work directory from the
// given YAML contents.
func (s *Session) PrepareActiveStateYAML(contents string) {
	require.NoError(s.T, fileutils.WriteFile(filepath.Join(s.Dirs.Work, constants.ConfigFileName), []byte(contents)))
}

func (s *Session) PrepareBuildScript(contents string) {
	require.NoError(s.T, fileutils.WriteFile(filepath.Join(s.Dirs.Work, constants.BuildScriptFileName), []byte(contents)))
}

func (s *Session) PrepareCommitIdFile(commitID string) {
	pjfile, err := projectfile.Parse(filepath.Join(s.Dirs.Work, constants.ConfigFileName))
	require.NoError(s.T, err)
	require.NoError(s.T, pjfile.SetLegacyCommit(commitID))
}

// CommitID is used to grab the current commit ID for the project in our working directory.
// For integration tests you should use this function instead of localcommit.Get() and pjfile.LegacyCommitID() as it
// is guaranteed to give a fresh result from disk, whereas the ones above use caching which tests don't like.
func (s *Session) CommitID() string {
	pjfile, err := projectfile.Parse(filepath.Join(s.Dirs.Work, constants.ConfigFileName))
	require.NoError(s.T, err)
	return pjfile.LegacyCommitID()
}

// PrepareEmptyProject creates a checkout of the empty ActiveState-CLI/Empty project without using
// `state checkout`.
func (s *Session) PrepareEmptyProject() {
	s.PrepareActiveStateYAML(fmt.Sprintf("project: https://%s/%s", constants.DefaultAPIHost, "ActiveState-CLI/Empty"))
	s.PrepareCommitIdFile("6d79f2ae-f8b5-46bd-917a-d4b2558ec7b8")
}

// PrepareProject creates a very simple activestate.yaml file for the given org/project and, if a
// commit ID is given, an .activestate/commit file.
func (s *Session) PrepareProject(namespace, commitID string) {
	s.PrepareActiveStateYAML(fmt.Sprintf("project: https://%s/%s", constants.DefaultAPIHost, namespace))
	if commitID != "" {
		s.PrepareCommitIdFile(commitID)
	}
}

func (s *Session) PrepareProjectAndBuildScript(namespace, commitID string) {
	s.PrepareProject(namespace, commitID)
	bp := buildplanner.NewBuildPlannerModel(nil, s.cache)
	script, err := bp.GetBuildScript(commitID)
	require.NoError(s.T, err)
	b, err := script.Marshal()
	require.NoError(s.T, err)
	err = fileutils.WriteFile(filepath.Join(s.Dirs.Work, constants.BuildScriptFileName), b)
	require.NoError(s.T, err)
}

// PrepareFile writes a file to path with contents, expecting no error
func (s *Session) PrepareFile(path, contents string) {
	errMsg := fmt.Sprintf("cannot setup file %q", path)

	contents = strings.TrimSpace(contents)

	err := os.MkdirAll(filepath.Dir(path), 0770)
	require.NoError(s.T, err, errMsg)

	bs := append([]byte(contents), '\n')

	err = os.WriteFile(path, bs, 0660)
	require.NoError(s.T, err, errMsg)
}

// LoginAsPersistentUser is a common test case after which an integration test user should be logged in to the platform
func (s *Session) LoginAsPersistentUser() {
	p := s.SpawnWithOpts(
		OptArgs(tagsuite.Auth, "--username", PersistentUsername, "--password", PersistentPassword),
		// as the command line includes a password, we do not print the executed command, so the password does not get logged
		OptHideArgs(),
	)

	p.Expect("logged in", termtest.OptExpectTimeout(defaultTimeout))
	p.ExpectExitCode(0)
}

func (s *Session) LogoutUser() {
	p := s.Spawn(tagsuite.Auth, "logout")
	p.ExpectExitCode(0)
}

// NotifyProjectCreated indicates that the given project was created on the Platform and needs to
// be deleted when the session is closed.
func (s *Session) NotifyProjectCreated(org, name string) {
	s.createdProjects = append(s.createdProjects, project.NewNamespace(org, name, ""))
}

const deleteUUIDProjects = "__delete_uuid_projects" // some unique project name

// DeleteUUIDProjects indicates that all projects with UUID names (i.e. autogenerated) for the given
// org should be deleted when the session is closed.
// This should not be called from generic integration tests. Use NotifyProjectCreated() instead,
// because there could be race conditions if multiple platforms are creating and using UUID
// projects.
func (s *Session) DeleteUUIDProjects(org string) {
	s.NotifyProjectCreated(org, deleteUUIDProjects)
}

func (s *Session) DebugMessage(prefix string) string {
	var sectionStart, sectionEnd string
	sectionStart = "\n=== "
	sectionEnd = "\n/==\n"
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		sectionStart = "##[group]"
		sectionEnd = "##[endgroup]"
	}

	if prefix != "" {
		prefix = prefix + "\n"
	}

	output := []string{}
	for _, spawn := range s.spawned {
		name := spawn.Cmd().String()
		if spawn.opts.HideCmdArgs {
			name = spawn.Cmd().Path
		}
		out := spawn.Output()
		if strings.Contains(out, "panic") || strings.Contains(out, "goroutine") {
			// If we encountered a panic it's unlikely the snapshot has enough information to be useful, so in this
			// case we include the full output. Which we don't normally do as it is just the dump of pty data, and
			// tends to be overly verbose and difficult to grok.
			output = append(output, fmt.Sprintf("Snapshot for Cmd '%s':\n%s", name, strings.TrimSpace(out)))
		} else {
			output = append(output, fmt.Sprintf("Snapshot for Cmd '%s':\n%s", name, strings.TrimSpace(spawn.Snapshot())))
		}
	}

	logs := []string{}
	for name, log := range s.DebugLogs() {
		logs = append(logs, fmt.Sprintf("Log for '%s':\n%s", name, log))
	}

	v, err := strutils.ParseTemplate(`
{{.Prefix}}Stack:
{{.Stacktrace}}
{{range $value := .Outputs}}
{{$.A}}{{$value}}{{$.Z}}
{{end}}
{{range $value := .Logs}}
{{$.A}}{{$value}}{{$.Z}}
{{else}}
No logs
{{end}}
`, map[string]interface{}{
		"Prefix":     prefix,
		"Stacktrace": stacktrace.Get().String(),
		"Outputs":    output,
		"Logs":       logs,
		"A":          sectionStart,
		"Z":          sectionEnd,
	}, nil)
	if err != nil {
		s.T.Fatalf("Parsing template failed: %s", errs.JoinMessage(err))
	}

	return v
}

// Close removes the temporary directory unless RetainDirs is specified
func (s *Session) Close() error {
	// stop service if it exists
	if fileutils.TargetExists(s.SvcExe) {
		cp := s.SpawnCmd(s.SvcExe, "stop")
		cp.ExpectExitCode(0)
	}

	cfg, err := config.NewCustom(s.Dirs.Config, singlethread.New(), true)
	require.NoError(s.T, err, "Could not read e2e session configuration: %s", errs.JoinMessage(err))

	if !s.retainDirs {
		defer s.Dirs.Close()
	}

	s.spawned = []*SpawnedCmd{}

	if os.Getenv("PLATFORM_API_TOKEN") == "" {
		s.T.Log("PLATFORM_API_TOKEN env var not set, not running suite tear down")
		return nil
	}

	auth := authentication.New(cfg)

	if os.Getenv(constants.APIHostEnvVarName) == "" {
		err := os.Setenv(constants.APIHostEnvVarName, constants.DefaultAPIHost)
		if err != nil {
			return err
		}
		defer func() {
			os.Unsetenv(constants.APIHostEnvVarName)
		}()
	}

	err = auth.AuthenticateWithModel(&mono_models.Credentials{
		Token: os.Getenv("PLATFORM_API_TOKEN"),
	})
	if err != nil {
		return err
	}

	if len(s.createdProjects) > 0 && s.createdProjects[0].Project == deleteUUIDProjects {
		org := s.createdProjects[0].Owner
		s.createdProjects = make([]*project.Namespaced, 0) // reset
		// When deleting UUID projects, only do it on one platform in order to avoid race conditions.
		if runtime.GOOS == "linux" {
			projects, err := getProjects(org, auth)
			if err != nil {
				s.T.Errorf("Could not fetch projects: %v", errs.JoinMessage(err))
			}
			for _, proj := range projects {
				if strfmt.IsUUID(proj.Name) {
					s.NotifyProjectCreated(org, proj.Name)
				}
			}
		}
	}

	for _, proj := range s.createdProjects {
		err := model.DeleteProject(proj.Owner, proj.Project, auth)
		if err != nil {
			s.T.Errorf("Could not delete project %s: %v", proj.Project, errs.JoinMessage(err))
		}
	}

	for _, user := range s.users {
		err := cleanUser(s.T, user, auth)
		if err != nil {
			s.T.Errorf("Could not delete user %s: %v", user, errs.JoinMessage(err))
		}
	}

	// Add back the release state tool installation to the bash RC file.
	// This was done on session creation to ensure that the release state tool
	// does not appear on the PATH when a new subshell is started. This is a
	// workaround to be addressed in: https://activestatef.atlassian.net/browse/DX-2285
	if runtime.GOOS != "windows" {
		installPath, err := installation.InstallPathForChannel("release")
		if err != nil {
			s.T.Errorf("Could not get install path: %v", errs.JoinMessage(err))
		}
		binDir := filepath.Join(installPath, "bin")

		ss := bash.SubShell{}
		err = ss.WriteUserEnv(cfg, map[string]string{"PATH": binDir}, sscommon.InstallID, false)
		if err != nil {
			s.T.Errorf("Could not clean user env: %v", errs.JoinMessage(err))
		}
	}

	if !s.ignoreLogErrors {
		s.detectLogErrors()
	}

	return nil
}

func (s *Session) InstallerLog() string {
	logDir := filepath.Join(s.Dirs.Config, "logs")
	if !fileutils.DirExists(logDir) {
		return ""
	}
	files, err := fileutils.ListDirSimple(logDir, false)
	if err != nil {
		return fmt.Sprintf("Could not list log dir: %v", err)
	}
	for _, file := range files {
		if !strings.HasPrefix(filepath.Base(file), "state-installer") {
			continue
		}
		b := fileutils.ReadFileUnsafe(file)
		return string(b) + "\n\nCurrent time: " + time.Now().String()
	}

	return fmt.Sprintf("Could not find state-installer log, checked under %s, found: \n, files: \n%v\n", logDir, files)
}

func (s *Session) SvcLog() string {
	logDir := filepath.Join(s.Dirs.Config, "logs")
	if !fileutils.DirExists(logDir) {
		return ""
	}
	files, err := fileutils.ListDirSimple(logDir, false)
	if err != nil {
		return fmt.Sprintf("Could not list log dir: %v", err)
	}
	lines := []string{}
	for _, file := range files {
		if !strings.HasPrefix(filepath.Base(file), "state-svc") {
			continue
		}
		b := fileutils.ReadFileUnsafe(file)
		lines = append(lines, filepath.Base(file)+":"+strings.Split(string(b), "\n")[0])
		if !strings.Contains(string(b), fmt.Sprintf("state-svc%s foreground", osutils.ExeExtension)) {
			continue
		}

		return string(b) + "\n\nCurrent time: " + time.Now().String()
	}

	return fmt.Sprintf("Could not find state-svc log, checked under %s, found: \n%v\n, files: \n%v\n", logDir, lines, files)
}

func (s *Session) LogFiles() []string {
	result := []string{}
	logDir := filepath.Join(s.Dirs.Config, "logs")
	if !fileutils.DirExists(logDir) {
		return result
	}

	err := filepath.WalkDir(logDir, func(path string, f fs.DirEntry, err error) error {
		if err != nil {
			panic(err)
		}
		if f.IsDir() {
			return nil
		}

		result = append(result, path)
		return nil
	})
	if err != nil {
		fmt.Printf("Error walking log dir: %v", err)
	}

	// Sort by filename timestamp (filenames are `[executable]-[processid]-[timestamp].log`)
	slices.SortFunc(result, func(a, b string) int {
		aa := strings.Split(a, "-")
		bb := strings.Split(b, "-")
		return strings.Compare(bb[len(bb)-1], aa[len(aa)-1])
	})

	return result
}

func (s *Session) DebugLogs() map[string]string {
	result := map[string]string{}

	logDir := filepath.Join(s.Dirs.Config, "logs")
	if !fileutils.DirExists(logDir) {
		return result
	}

	for _, path := range s.LogFiles() {
		result[filepath.Base(path)] = string(fileutils.ReadFileUnsafe(path))
	}

	return result
}

func (s *Session) DebugLogsDump() string {
	logs := s.DebugLogs()

	if len(logs) == 0 {
		return "No logs found in " + filepath.Join(s.Dirs.Config, "logs")
	}

	var sectionStart, sectionEnd string
	sectionStart = "\n=== "
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		sectionStart = "##[group]"
		sectionEnd = "##[endgroup]"
	}

	result := "Logs:\n"
	for name, log := range logs {
		result += fmt.Sprintf("%s%s:\n%s%s\n", sectionStart, name, log, sectionEnd)
	}

	return result
}

// IgnoreLogErrors disables log error checking after the session closes.
// Normally, logged errors automatically cause test failures, so calling this is needed for tests
// with expected errors.
func (s *Session) IgnoreLogErrors() {
	s.ignoreLogErrors = true
}

var errorOrPanicRegex = regexp.MustCompile(`(?:\[ERR |\[CRT |Panic:)`)

func (s *Session) detectLogErrors() {
	var sectionStart, sectionEnd string
	sectionStart = "\n=== "
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		sectionStart = "##[group]"
		sectionEnd = "##[endgroup]"
	}
	for _, path := range s.LogFiles() {
		if !strings.HasPrefix(filepath.Base(path), "state-") {
			continue
		}
		if contents := string(fileutils.ReadFileUnsafe(path)); errorOrPanicRegex.MatchString(contents) {
			s.T.Errorf(s.DebugMessage(fmt.Sprintf("%sFound error and/or panic in log file %s\nIf this was expected, call session.IgnoreLogErrors() to avoid this check\nLog contents:\n%s%s",
				sectionStart, path, contents, sectionEnd)))
		}
	}
}

func (s *Session) SetupRCFile() {
	if runtime.GOOS == "windows" {
		return
	}
	s.T.Setenv("HOME", s.Dirs.HomeDir)
	defer s.T.Setenv("HOME", os.Getenv("HOME"))

	cfg, err := config.New()
	require.NoError(s.T, err)

	s.SetupRCFileCustom(subshell.New(cfg))
}

func (s *Session) SetupRCFileCustom(subshell subshell.SubShell) {
	if runtime.GOOS == "windows" {
		return
	}

	rcFile, err := subshell.RcFile()
	require.NoError(s.T, err)

	if fileutils.TargetExists(filepath.Join(s.Dirs.HomeDir, filepath.Base(rcFile))) {
		err = fileutils.CopyFile(rcFile, filepath.Join(s.Dirs.HomeDir, filepath.Base(rcFile)))
	} else {
		err = fileutils.Touch(filepath.Join(s.Dirs.HomeDir, filepath.Base(rcFile)))
	}
	require.NoError(s.T, err)
}

func RunningOnCI() bool {
	return condition.OnCI()
}
