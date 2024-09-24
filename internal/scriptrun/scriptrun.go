package scriptrun

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/process"
	"github.com/ActiveState/cli/internal/runbits/runtime"
	"github.com/ActiveState/cli/internal/runbits/runtime/trigger"
	"github.com/ActiveState/cli/internal/scriptfile"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

type primeable interface {
	primer.Auther
	primer.Outputer
	primer.Projecter
	primer.Subsheller
	primer.Configurer
	primer.Analyticer
	primer.SvcModeler
}

// ScriptRun manages the context required to run a script.
type ScriptRun struct {
	prime primeable
	// The remainder is redundant with the above. Refactoring this will follow in a later story so as not to blow
	// up the one that necessitates adding the primer at this level.
	// https://activestatef.atlassian.net/browse/DX-2869

	auth      *authentication.Auth
	out       output.Outputer
	sub       subshell.SubShell
	project   *project.Project
	cfg       *config.Instance
	analytics analytics.Dispatcher
	svcModel  *model.SvcModel

	venvPrepared bool
	venvExePath  string
}

// New returns a pointer to a prepared instance of ScriptRun.
func New(prime primeable) *ScriptRun {
	return &ScriptRun{
		prime,
		prime.Auth(),
		prime.Output(),
		prime.Subshell(),
		prime.Project(),
		prime.Config(),
		prime.Analytics(),
		prime.SvcModel(),

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
func (s *ScriptRun) PrepareVirtualEnv() (rerr error) {
	rt, err := runtime_runbit.Update(s.prime, trigger.TriggerScript, runtime_runbit.WithoutHeaders())
	if err != nil {
		return locale.WrapError(err, "err_activate_runtime", "Could not initialize a runtime for this project.")
	}

	venv := virtualenvironment.New(rt)

	projDir := filepath.Dir(s.project.Source().Path())
	env, err := venv.GetEnv(true, true, projDir, s.project.Namespace().String())
	if err != nil {
		return errs.Wrap(err, "Could not get venv environment")
	}
	err = s.sub.SetEnv(env)
	if err != nil {
		return locale.WrapError(err, "err_subshell_setenv")
	}

	// search the "clean" path first (PATHS that are set by venv)
	env, err = venv.GetEnv(false, true, "", "")
	if err != nil {
		return errs.Wrap(err, "Could not get venv environment")
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
	var attempted3rdParty []string
	for _, l := range script.Languages() {
		execPath := l.Executable().Name()
		searchPath := s.venvExePath
		if l.Executable().CanUseThirdParty() {
			searchPath = searchPath + string(os.PathListSeparator) + os.Getenv("PATH")
		}

		logging.Debug("Checking for %s on %s", execPath, searchPath)
		if PathProvidesExec(searchPath, execPath) {
			lang = l
			logging.Debug("Found %s", execPath)
			break
		}
		attempted = append(attempted, l.String())
		if l.Executable().CanUseThirdParty() {
			attempted3rdParty = append(attempted3rdParty, l.Executable().Filename())
		}
	}

	if script.Standalone() && !lang.Executable().CanUseThirdParty() {
		return locale.NewInputError("error_state_run_standalone_conflict")
	}

	if lang == language.Unknown {
		if len(attempted) > 0 {
			err := locale.NewInputError(
				"err_run_unknown_language_fallback",
				"The language for this script is not supported or not available on your system. Attempted script execution with: {{.V0}}. Please configure the 'language' field with an available option (one, or more, of: {{.V1}})",
				strings.Join(attempted, ", "),
				strings.Join(language.RecognizedNames(), ", "),
			)
			if len(attempted3rdParty) == 0 {
				return err
			}
			return errs.AddTips(err, locale.Tl("unknown_language_check_path", "Please ensure that one of these executables is on your PATH: [ACTIONABLE]{{.V0}}[/RESET]", strings.Join(attempted3rdParty, ", ")))
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
				"Script execution fell back to '{{.V0}}' after '{{.V1}}' was not detected in your project or system. Please ensure your script is compatible with one, or more, of: {{.V0}}, {{.V1}}",
				lang.String(),
				strings.Join(attempted, ", "),
			)
		}
		return errs.AddTips(
			locale.WrapInputError(err, "err_script_run", "Your script failed to execute: {{.V0}}.", err.Error()),
			locale.Tl("script_run_tip", "Edit the script '[ACTIONABLE]{{.V0}}[/RESET]' in your [ACTIONABLE]activestate.yaml[/RESET].", script.Name()))
	}
	return nil
}

func PathProvidesExec(path, exec string) bool {
	return osutils.FindExeInside(exec, path) != ""
}
