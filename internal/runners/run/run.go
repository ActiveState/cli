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
	"github.com/ActiveState/cli/internal/output/txtstyle"
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
	// FailStandaloneConflict indicates when a script is run standalone, but unable to be so
	FailStandaloneConflict = failures.Type("run.fail.standaloneconflict", failures.FailUser)
	// FailExecNotFound indicates when the builtin language exec is not available
	FailExecNotFound = failures.Type("run.fail.execnotfound", failures.FailUser)
)

// Run contains the run execution context.
type Run struct {
	out      output.Outputer
	proj     *project.Project
	subshell subshell.SubShell
}

type primeable interface {
	primer.Outputer
	primer.Projecter
	primer.Subsheller
}

// New constructs a new instance of Run.
func New(prime primeable) *Run {
	return &Run{
		prime.Output(),
		prime.Project(),
		prime.Subshell(),
	}
}

// Run runs the Run run runner.
func (r *Run) Run(name string, args []string) error {
	return run(r.out, r.subshell, r.proj, name, args)
}

func run(out output.Outputer, subs subshell.SubShell, proj *project.Project, name string, args []string) error {
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

	out.Notice(txtstyle.NewTitle(locale.Tl("run_script_title", "Running Script: [ACTIONABLE]{{.V0}}[/RESET]", script.Name())))

	if authentication.Get().Authenticated() {
		checker.RunCommitsBehindNotifier(out)
	}

	// venvExePath stores a virtual environment's PATH value. If the script
	// requires activation this is the PATH we should be searching for
	// executables in.
	venvExePath := os.Getenv("PATH")

	// Activate the state if needed.
	if !script.Standalone() && !subshell.IsActivated() {
		out.Notice(output.Heading(locale.Tl("notice", "Notice")))
		out.Notice(locale.T("info_state_run_activating_state"))
		runtime, err := runtime.NewRuntime(proj.Source().Path(), proj.CommitUUID(), proj.Owner(), proj.Name(), runbits.NewRuntimeMessageHandler(out))
		if err != nil {
			return locale.WrapError(err, "err_run_runtime_init", "Failed to initialize runtime.")
		}
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

		// search the "clean" path first (PATHS that are set by venv)
		env, err = venv.GetEnv(false, "")
		if err != nil {
			return err
		}
		venvExePath = env["PATH"]
	}

	lang := language.Unknown
	if len(script.Languages()) == 0 {
		warning := locale.Tl(
			"run_warn_deprecated_script_without_language",
			"Scripts without a defined language currently fall back to using the default shell for your platform. This fallback mechanic will soon stop working and a language will need to be explicitly defined for each script. Please configure the '[ACTIONABLE]language[/RESET]' field with a valid option (one of [ACTIONABLE]{{.V0}}[/RESET])",
			strings.Join(language.RecognizedNames(), ", "),
		)
		out.Notice(output.Heading(locale.Tl("deprecation_warning", "Deprecation Warning!")))
		out.Notice(warning)

		lang = language.MakeByShell(subs.Shell())
	}

	var attempted []string
	for _, l := range script.Languages() {
		var execPath string
		var searchPath string
		if l.Executable().Available() {
			execPath = l.Executable().Name()
			searchPath = venvExePath
		} else {
			execPath = l.String()
			searchPath = os.Getenv("PATH")
		}

		if l.Executable().Builtin() && rt.GOOS == "windows" {
			execPath = execPath + ".exe"
		}

		if pathProvidesExec(searchPath, execPath) {
			lang = l
			break
		}
		attempted = append(attempted, l.String())
	}

	if script.Standalone() && !lang.Executable().Builtin() {
		return FailStandaloneConflict.New("error_state_run_standalone_conflict")
	}

	if lang == language.Unknown {
		if len(attempted) > 0 {
			return locale.NewInputError(
				"err_run_unknown_language_fallback",
				"The language for this script is not supported or not available on your system. Attempted script execution with: {{.V0}}. Please configure the 'language' field with an available option (one, or more, of: {{.V1}})",
				strings.Join(attempted, ", "),
				strings.Join(language.RecognizedNames(), ", "),
			)
		}
		return locale.NewInputError(
			"err_run_unknown_language",
			"The language for this script is not supported or not available on your system. Please configure the 'language' field with a valid option (one, or more, of: {{.V0}})", strings.Join(language.RecognizedNames(), ", "),
		)
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

	out.Notice(output.Heading(locale.Tl("script_output", "Script Output")))
	// ignore code for now, passing via failure
	err = subs.Run(sf.Filename(), args...)
	if err != nil {
		if len(attempted) > 0 {
			return locale.WrapInputError(
				err,
				"err_run_script",
				"Script execution fell back to {{.V0}} after {{.V1}} was not detected in your project or system. Please ensure your script is compatible with one, or more, of: {{.V0}}, {{.V1}}",
				lang.String(),
				strings.Join(attempted, ", "),
			)
		}
		return err
	}
	return nil
}

func configCachePath() string {
	if rt.GOOS == "darwin" { // runtime loading is not yet supported in darwin systems
		return "" // empty string value will skip path filtering in subsequent logic
	}
	return config.CachePath()
}

func pathProvidesExec(path, exec string) bool {
	paths := splitPath(path)
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
