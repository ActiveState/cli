package run

import (
	"strings"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/scriptrun"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/pkg/cmdlets/checker"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// Run contains the run execution context.
type Run struct {
	auth      *authentication.Auth
	out       output.Outputer
	proj      *project.Project
	subshell  subshell.SubShell
	cfg       *config.Instance
	svcModel  *model.SvcModel
	analytics analytics.Dispatcher
}

type primeable interface {
	primer.Auther
	primer.Outputer
	primer.Projecter
	primer.Subsheller
	primer.Configurer
	primer.SvcModeler
	primer.Analyticer
}

// New constructs a new instance of Run.
func New(prime primeable) *Run {
	return &Run{
		prime.Auth(),
		prime.Output(),
		prime.Project(),
		prime.Subshell(),
		prime.Config(),
		prime.SvcModel(),
		prime.Analytics(),
	}
}

// Run runs the Run run runner.
func (r *Run) Run(name string, args []string) error {
	logging.Debug("Execute")

	checker.RunUpdateNotifier(r.svcModel, r.out)

	if r.proj == nil {
		return locale.NewInputError("err_no_project")
	}

	if name == "" {
		return locale.NewError("error_state_run_undefined_name")
	}

	r.out.Notice(output.Title(locale.Tl("run_script_title", "Running Script: [ACTIONABLE]{{.V0}}[/RESET]", name)))

	if authentication.LegacyGet().Authenticated() {
		checker.RunCommitsBehindNotifier(r.proj, r.out)
	}

	script := r.proj.ScriptByName(name)
	if script == nil {
		return locale.NewInputError("error_state_run_unknown_name", "", name)
	}

	scriptrunner := scriptrun.New(r.auth, r.out, r.subshell, r.proj, r.cfg, r.analytics, r.svcModel)
	if !script.Standalone() && scriptrunner.NeedsActivation() {
		if err := scriptrunner.PrepareVirtualEnv(); err != nil {
			return locale.WrapError(err, "err_script_run_preparevenv", "Could not prepare virtual environment.")
		}
	}

	if len(script.Languages()) == 0 {
		r.out.Notice(output.Heading(locale.Tl("deprecation_warning", "Deprecation Warning!")))
		r.out.Notice(locale.Tl(
			"run_warn_deprecated_script_without_language",
			"Scripts without a defined language currently fall back to using the default shell for your platform. This fallback mechanic will soon stop working and a language will need to be explicitly defined for each script. Please configure the '[ACTIONABLE]language[/RESET]' field with a valid option (one of [ACTIONABLE]{{.V0}}[/RESET])",
			strings.Join(language.RecognizedNames(), ", ")))
	}

	r.out.Notice(output.Heading(locale.Tl("script_output", "Script Output")))
	return scriptrunner.Run(script, args)
}
