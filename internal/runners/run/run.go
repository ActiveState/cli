package run

import (
	"strings"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits/checker"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/internal/scriptrun"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// Run contains the run execution context.
type Run struct {
	prime primeable
	// The remainder is redundant with the above. Refactoring this will follow in a later story so as not to blow
	// up the one that necessitates adding the primer at this level.
	// https://activestatef.atlassian.net/browse/DX-2869
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
		prime,
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

	if r.proj == nil {
		return rationalize.ErrNoProject
	}

	r.out.Notice(locale.Tr("operating_message", r.proj.NamespaceString(), r.proj.Dir()))

	if name == "" {
		return locale.NewError("error_state_run_undefined_name")
	}

	r.out.Notice(output.Title(locale.Tl("run_script_title", "Running Script: [ACTIONABLE]{{.V0}}[/RESET]", name)))

	if r.auth.Authenticated() {
		checker.RunCommitsBehindNotifier(r.proj, r.out, r.auth, r.cfg)
	}

	script, err := r.proj.ScriptByName(name)
	if err != nil {
		return errs.Wrap(err, "Could not get script")
	}
	if script == nil {
		return locale.NewInputError("error_state_run_unknown_name", "", name)
	}

	scriptrunner := scriptrun.New(r.prime)
	if !script.Standalone() && scriptrunner.NeedsActivation() {
		if err := scriptrunner.PrepareVirtualEnv(); err != nil {
			return locale.WrapError(err, "err_script_run_preparevenv", "Could not prepare virtual environment.")
		}
	}

	if len(script.Languages()) == 0 {
		r.out.Notice(output.Title(locale.Tl("deprecation_warning", "Deprecation Warning!")))
		r.out.Notice(locale.Tl(
			"run_warn_deprecated_script_without_language",
			"Scripts without a defined language currently fall back to using the default shell for your platform. This fallback mechanic will soon stop working and a language will need to be explicitly defined for each script. Please configure the '[ACTIONABLE]language[/RESET]' field with a valid option (one of [ACTIONABLE]{{.V0}}[/RESET])",
			strings.Join(language.RecognizedNames(), ", "),
		))
	}

	r.out.Notice(output.Title(locale.Tl("script_output", "Script Output")))
	return scriptrunner.Run(script, args)
}
