package run

import (
	"os"
	"path/filepath"
	rt "runtime"
	"strings"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/internal/scriptfile"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/ActiveState/cli/pkg/cmdlets/checker"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/project"
)

var (
	// FailScriptNotDefined indicates the user provided a script name that is not defined
	FailScriptNotDefined = failures.Type("run.fail.scriptnotfound", failures.FailUser)
	// FailStandalonConflict indicates when a script is run standalone, but unable to be so
	FailStandalonConflict = failures.Type("run.fail.standaloneconflict", failures.FailUser)
	// FailExecNotFound indicates when the builtin language exec is not available
	FailExecNotFound = failures.Type("run.fail.execnotfound", failures.FailUser)
)

// Run contains the run execution context.
type Run struct {
	out      output.Outputer
	subshell subshell.SubShell
	project  *project.Project
}

type primeable interface {
	primer.Outputer
	primer.Subsheller
	primer.Projecter
}

// New constructs a new instance of Run.
func New(prime primeable) *Run {
	return &Run{
		prime.Output(),
		prime.Subshell(),
		prime.Project(),
	}
}

// Run runs the Run run runner.
func (r *Run) Run(name string, args []string) error {
	return run(r.out, r.subshell, r.project, name, args)
}

func run(out output.Outputer, subs subshell.SubShell, proj *project.Project, name string, args []string) error {
	if authentication.Get().Authenticated() {
		checker.RunCommitsBehindNotifier(out)
	}

	logging.Debug("Execute")

	if name == "" {
		return failures.FailUserInput.New("error_state_run_undefined_name")
	}

	// Determine which project script to run based on the given script name.
	script := proj.ScriptByName(name)
	if script == nil {
		fail := FailScriptNotDefined.New(
			locale.T("error_state_run_unknown_name", map[string]string{"Name": name}),
		)
		return fail
	}

	lang := script.Language()
	if !lang.Recognized() {
		warning := locale.Tl(
			"run_warn_deprecated_script_without_language",
			"[YELLOW]DEPRECATION WARNING: Scripts without a defined language currently fall back to using  the default shell for your platform. This fallback mechanic will soon stop working and a language will need to be explicitly defined for each script. Please configure the 'language' field with a valid option (one of {{.V0}})[/RESET]",
			strings.Join(language.RecognizedNames(), ", "),
		)
		out.Notice(warning)

		lang = language.MakeByShell(subs.Shell())
	}

	langExec := lang.Executable()
	if script.Standalone() && !langExec.Builtin() {
		return FailStandalonConflict.New("error_state_run_standalone_conflict")
	}

	envPath := os.Getenv("PATH")

	// Activate the state if needed.
	if !script.Standalone() && !subshell.IsActivated() {
		out.Notice(locale.T("info_state_run_activating_state"))
		runtime := runtime.NewRuntime(proj.CommitUUID(), proj.Owner(), proj.Name(), runbits.NewRuntimeMessageHandler(out))
		venv := virtualenvironment.New(runtime)

		if fail := venv.Activate(); fail != nil {
			logging.Errorf("Unable to activate state: %s", fail.Error())
			return fail.WithDescription("error_state_run_activate")
		}

		env, err := venv.GetEnv(true, filepath.Dir(proj.Source().Path()))
		if err != nil {
			return err
		}
		subs.SetEnv(env)

		// get the "clean" path (only PATHS that are set by venv)
		env, err = venv.GetEnv(false, "")
		if err != nil {
			return err
		}
		envPath = env["PATH"]
	}

	if !langExec.Builtin() && !pathProvidesExec(configCachePath(), langExec.Name(), envPath) {
		return FailExecNotFound.New("error_state_run_unknown_exec")
	}

	scriptBlock, err := script.Value()
	if err != nil {
		return locale.WrapError(err, "err_run_scriptval", "Could not get script value.")
	}

	sf, fail := scriptfile.New(lang, script.Name(), scriptBlock)
	if fail != nil {
		return fail.WithDescription("error_state_run_setup_scriptfile")
	}
	defer sf.Clean()

	out.Notice(locale.Tr("info_state_run_running", script.Name(), script.Source().Path()))
	// ignore code for now, passing via failure
	return subs.Run(sf.Filename(), args...)
}

func configCachePath() string {
	if rt.GOOS == "darwin" { // runtime loading is not yet supported in darwin systems
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

	if rt.GOOS == "windows" {
		return f.Mode()&0400 != 0
	}

	return f.Mode()&0110 != 0
}
