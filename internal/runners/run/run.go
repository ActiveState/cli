package run

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/scriptfile"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/ActiveState/cli/pkg/cmdlets/checker"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

var (
	// FailScriptNotDefined indicates the user provided a script name that is not defined
	FailScriptNotDefined = failures.Type("run.fail.scriptnotfound", failures.FailUser)
	// FailStandalonConflict indicates when a script is run standalone, but unable to be so
	FailStandalonConflict = failures.Type("run.fail.standaloneconflict", failures.FailUser)
	// FailExecNotFound indicates when the builtin language exec is not available
	FailExecNotFound = failures.Type("run.fail.execnotfound", failures.FailUser)
)

type Run struct {
}

func New() *Run {
	return &Run{}
}

// Execute the run command.
func (r *Run) Run(name string, args []string) error {
	return run(name, args)
}

func run(name string, args []string) error {
	if authentication.Get().Authenticated() {
		checker.RunCommitsBehindNotifier()
	}

	logging.Debug("Execute")

	if name == "" {
		return failures.FailUserInput.New("error_state_run_undefined_name")
	}

	// Determine which project script to run based on the given script name.
	script := project.Get().ScriptByName(name)
	if script == nil {
		fail := FailScriptNotDefined.New(
			locale.T("error_state_run_unknown_name", map[string]string{"Name": name}),
		)
		return fail
	}

	subs, fail := subshell.Get()
	if fail != nil {
		return fail.WithDescription("error_state_run_no_shell")
	}

	lang := script.Language()
	if !lang.Recognized() || !lang.Executable().Available() {
		lang = language.MakeByShell(subs.Shell())
	}

	langExec := lang.Executable()
	if script.Standalone() && !langExec.Builtin() {
		return FailStandalonConflict.New("error_state_run_standalone_conflict")
	}

	path := os.Getenv("PATH")

	// Activate the state if needed.
	if !script.Standalone() && !subshell.IsActivated() {
		print.Info(locale.T("info_state_run_activating_state"))
		venv := virtualenvironment.Init()
		venv.OnDownloadArtifacts(func() { print.Line(locale.T("downloading_artifacts")) })
		venv.OnInstallArtifacts(func() { print.Line(locale.T("installing_artifacts")) })

		if err := venv.Activate(); err != nil {
			logging.Errorf("Unable to activate state: %s", fail.Error())
			return locale.WrapError(err, "error_state_run_activate", `Unable to activate a state for running the script in. Try manually running "state activate" first.`)
		}

		env, err := venv.GetEnv(true, filepath.Dir(projectfile.Get().Path()))
		if err != nil {
			return err
		}
		subs.SetEnv(env)

		// get the "clean" path (only PATHS that are set by venv)
		env, err = venv.GetEnv(false, "")
		if err != nil {
			return err
		}
		path = env["PATH"]
	}

	if !langExec.Builtin() && !pathProvidesExec(configCachePath(), langExec.Name(), path) {
		return FailExecNotFound.New("error_state_run_unknown_exec")
	}

	// Run the script.
	scriptBlock := project.Expand(script.Value())
	sf, fail := scriptfile.New(lang, script.Name(), scriptBlock)
	if fail != nil {
		return fail.WithDescription("error_state_run_setup_scriptfile")
	}
	defer sf.Clean()

	print.Info(locale.Tr("info_state_run_running", script.Name(), script.Source().Path()))
	// ignore code for now, passing via failure
	return subs.Run(sf.Filename(), args...)
}

func configCachePath() string {
	if runtime.GOOS == "darwin" { // runtime loading is not yet supported in darwin systems
		return "" // empty string value will skip path filtering in subsequent logic
	}
	return config.CachePath()
}

func pathProvidesExec(filterByPath, exec, path string) bool {
	paths := splitPath(path)
	if filterByPath != "" {
		paths = filterPrefixed(filterByPath, paths)
	}
	paths = applySuffix(exec, paths)

	for _, p := range paths {
		if isExecutableFile(p) {
			return true
		}
	}
	return false
}

func splitPath(path string) []string {
	return strings.Split(path, string(os.PathListSeparator))
}

func filterPrefixed(prefix string, paths []string) []string {
	var ps []string
	for _, p := range paths {
		// Clean removes double slashes and relative path directories
		if strings.HasPrefix(filepath.Clean(p), filepath.Clean(prefix)) {
			ps = append(ps, p)
		}
	}
	return ps
}

func applySuffix(suffix string, paths []string) []string {
	for i, v := range paths {
		paths[i] = filepath.Join(v, suffix)
	}
	return paths
}

func isExecutableFile(name string) bool {
	f, err := os.Stat(name)
	if err != nil { // unlikely unless file does not exist
		return false
	}

	if runtime.GOOS == "windows" {
		return f.Mode()&0400 != 0
	}

	return f.Mode()&0110 != 0
}
