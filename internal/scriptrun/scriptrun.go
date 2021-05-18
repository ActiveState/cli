package scriptrun

import (
	"os"
	"path/filepath"
	rt "runtime"
	"strings"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/process"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/internal/scriptfile"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/platform/runtime/executor"
	"github.com/ActiveState/cli/pkg/project"
)

// ScriptRun manages the context required to run a script.
type ScriptRun struct {
	auth    *authentication.Auth
	out     output.Outputer
	sub     subshell.SubShell
	project *project.Project
	cfg     *config.Instance

	venvPrepared bool
	venvExePath  string
}

// New returns a pointer to a prepared instance of ScriptRun.
func New(auth *authentication.Auth, out output.Outputer, subs subshell.SubShell, proj *project.Project, cfg *config.Instance) *ScriptRun {
	return &ScriptRun{
		auth,
		out,
		subs,
		proj,
		cfg,

		false,

		// venvExePath stores a virtual environment's PATH value. If the script
		// requires activation this is the PATH we should be searching for
		// executables in.
		os.Getenv("PATH")}
}

// NeedsActivation indicates whether the underlying environment has been
// prepared and activated.
func (s *ScriptRun) NeedsActivation() bool {
	return !process.IsActivated(s.cfg) && !s.venvPrepared
}

// PrepareVirtualEnv sets up the relevant runtime and prepares the environment.
func (s *ScriptRun) PrepareVirtualEnv() error {
	rt, err := runtime.New(runtime.NewProjectTarget(s.project, s.cfg.CachePath(), nil))
	if err != nil {
		if !runtime.IsNeedsUpdateError(err) {
			return locale.WrapError(err, "err_activate_runtime", "Could not initialize a runtime for this project.")
		}
		if err := rt.Update(s.auth, runbits.DefaultRuntimeEventHandler(s.out)); err != nil {
			return locale.WrapError(err, "err_update_runtime", "Could not update runtime installation.")
		}
	}
	venv := virtualenvironment.New(rt)

	env, err := venv.GetEnv(true, true, filepath.Dir(s.project.Source().Path()))
	if err != nil {
		return err
	}
	s.sub.SetEnv(env)

	// search the "clean" path first (PATHS that are set by venv)
	env, err = venv.GetEnv(false, true, "")
	if err != nil {
		return err
	}
	s.venvExePath = env["PATH"]
	s.venvPrepared = true

	return nil
}

// Run executes the script after ensuring the environment is prepared.
func (s *ScriptRun) Run(script *project.Script, args []string) error {
	if s.project == nil {
		return locale.NewInputError("err_no_projectfile")
	}

	// Determine which project script to run based on the given script name.
	if script == nil {
		return locale.NewInputError("error_state_run_unknown_name", "Requested script does not exist.")
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
		execPath := l.Executable().Filename()
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
			return errs.AddTips(
				locale.NewInputError(
					"err_run_unknown_language_fallback",
					"The language for this script is not supported or not available on your system. Attempted script execution with: {{.V0}}. Please configure the 'language' field with an available option (one, or more, of: {{.V1}})",
					strings.Join(attempted, ", "),
					strings.Join(language.RecognizedNames(), ", "),
				),
				locale.Tl("unknown_language_check_path", "If the language is not provided by the project runtime, please ensure that the required executable is discoverable via your PATH environment variable"))
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

	sf, err := scriptfile.New(lang, script.Name(), scriptBlock)
	if err != nil {
		return locale.WrapError(err, "error_state_run_setup_scriptfile")
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
	// TODO: We want a better way to find the executable on Windows.
	// Follow up filed here: https://www.pivotaltracker.com/n/projects/2203557/stories/177934469
	if !fileutils.FileExists(name) {
		name = executor.NameForExe(name)
	}

	f, err := os.Stat(name)
	if err != nil { // unlikely unless file does not exist
		return false
	}

	if rt.GOOS == "windows" {
		return f.Mode()&0400 != 0
	}

	return f.Mode()&0110 != 0
}
