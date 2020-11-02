package scriptrun

import (
	"os"
	"path/filepath"
	rt "runtime"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/internal/scriptfile"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/project"
)

// ProjectHasScript is a helper function to determine if a project contains a
// script by name.
func ProjectHasScript(proj *project.Project, name string) bool {
	script := proj.ScriptByName(name)
	return script != nil
}

type ScriptRun struct {
	out     output.Outputer
	sub     subshell.SubShell
	project *project.Project

	venvPrepared bool
	venvExePath  string
}

func New(out output.Outputer, subs subshell.SubShell, proj *project.Project) *ScriptRun {
	return &ScriptRun{
		out,
		subs,
		proj,

		false,

		// venvExePath stores a virtual environment's PATH value. If the script
		// requires activation this is the PATH we should be searching for
		// executables in.
		os.Getenv("PATH")}
}

func (s *ScriptRun) NeedsActivation() bool {
	return !subshell.IsActivated() && !s.venvPrepared
}

func (s *ScriptRun) PrepareVirtualEnv() error {
	runtime, err := runtime.NewRuntime(s.project.Source().Path(), s.project.CommitUUID(), s.project.Owner(), s.project.Name(), runbits.NewRuntimeMessageHandler(s.out))
	if err != nil {
		return locale.WrapError(err, "err_run_runtime_init", "Failed to initialize runtime.")
	}
	venv := virtualenvironment.New(runtime)

	if fail := venv.Activate(); fail != nil {
		logging.Errorf("Unable to activate state: %s", fail.Error())
		return fail.WithDescription("error_state_run_activate")
	}

	env, err := venv.GetEnv(true, filepath.Dir(s.project.Source().Path()))
	if err != nil {
		return err
	}
	s.sub.SetEnv(env)

	// search the "clean" path first (PATHS that are set by venv)
	env, err = venv.GetEnv(false, "")
	if err != nil {
		return err
	}
	s.venvExePath = env["PATH"]
	s.venvPrepared = true

	return nil
}

func (s *ScriptRun) Run(script *project.Script, args []string) error {
	if s.project == nil {
		return locale.NewInputError("err_no_projectfile")
	}

	// Determine which project script to run based on the given script name.
	if script == nil {
		return locale.NewInputError("error_state_run_unknown_name", "Script does not exist: {{.V0}}.", script.Name())
	}

	// Activate the state if needed.
	if !script.Standalone() && s.NeedsActivation() {
		if err := s.PrepareVirtualEnv(); err != nil {
			return errs.Wrap(err, "Could not prepare virtual environment.")
		}
	}

	lang := language.Unknown
	if len(script.Languages()) == 0 {
		lang = language.MakeByShell(s.sub.Shell())
	}

	var attempted []string
	for _, l := range script.Languages() {
		execPath := l.Executable().Name()
		searchPath := s.venvExePath
		if l.Executable().CanUseThirdParty() {
			searchPath = searchPath + string(os.PathListSeparator) + os.Getenv("PATH")
		}

		logging.Debug("Checking for %s on %s", execPath, searchPath)
		if pathProvidesExec(searchPath, execPath) {
			lang = l
			logging.Debug("Found %s", execPath)
			break
		}
		attempted = append(attempted, l.String())
	}

	if script.Standalone() && !lang.Executable().CanUseThirdParty() {
		return locale.NewInputError("error_state_run_standalone_conflict")
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
		return locale.WrapError(fail, "error_state_run_setup_scriptfile")
	}
	defer sf.Clean()

	// ignore code for now, passing via failure
	err = s.sub.Run(sf.Filename(), args...)
	if err != nil {
		if len(attempted) > 0 {
			err = locale.WrapInputError(
				err,
				"err_run_script",
				"Script execution fell back to {{.V0}} after {{.V1}} was not detected in your project or system. Please ensure your script is compatible with one, or more, of: {{.V0}}, {{.V1}}",
				lang.String(),
				strings.Join(attempted, ", "),
			)
		}
		return errs.AddTips(
			locale.WrapError(err, "err_script_run", "Your script failed to execute: {{.V0}}.", err.Error()),
			locale.Tl("script_run_tip", "Edit the script '[ACTIONABLE]{{.V0}}[/RESET]' in your [ACTIONABLE]activestate.yaml[/RESET].", script.Name()))
	}
	return nil
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
