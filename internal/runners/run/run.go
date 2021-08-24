package run

import (
	"strings"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/scriptrun"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/svcmanager"
	"github.com/ActiveState/cli/pkg/cmdlets/checker"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/project"
)

// Run contains the run execution context.
type Run struct {
	auth     *authentication.Auth
	out      output.Outputer
	proj     *project.Project
	subshell subshell.SubShell
	cfg      *config.Instance
	svcMgr   *svcmanager.Manager
}

type primeable interface {
	primer.Auther
	primer.Outputer
	primer.Projecter
	primer.Subsheller
	primer.Configurer
	primer.Svcer
}

// New constructs a new instance of Run.
func New(prime primeable) *Run {
	return &Run{
		prime.Auth(),
		prime.Output(),
		prime.Project(),
		prime.Subshell(),
		prime.Config(),
		prime.SvcManager(),
	}
}

// Run runs the Run run runner.
func (r *Run) Run(name string, args []string) error {
	return run(r.auth, r.out, r.subshell, r.proj, r.svcMgr, r.cfg, name, args)
}

func run(auth *authentication.Auth, out output.Outputer, subs subshell.SubShell, proj *project.Project, svcMgr *svcmanager.Manager, cfg *config.Instance, name string, args []string) error {
	logging.Debug("Execute")

	checker.RunUpdateNotifier(svcMgr, cfg, out)

	if proj == nil {
		return locale.NewInputError("err_no_project")
	}

	if name == "" {
		return locale.NewError("error_state_run_undefined_name")
	}

	out.Notice(output.Title(locale.Tl("run_script_title", "Running Script: [ACTIONABLE]{{.V0}}[/RESET]", name)))

	if authentication.LegacyGet().Authenticated() {
		checker.RunCommitsBehindNotifier(proj, out)
	}

	script := proj.ScriptByName(name)
	if script == nil {
		return locale.NewInputError("error_state_run_unknown_name", "Script does not exist: {{.V0}}", name)
	}

	scriptrunner := scriptrun.New(auth, out, subs, proj, cfg)
	if !script.Standalone() && scriptrunner.NeedsActivation() {
		if err := scriptrunner.PrepareVirtualEnv(); err != nil {
			return locale.WrapError(err, "err_script_run_preparevenv", "Could not prepare virtual environment.")
		}
	}

	if len(script.Languages()) == 0 {
		out.Notice(output.Heading(locale.Tl("deprecation_warning", "Deprecation Warning!")))
		out.Notice(locale.Tl(
			"run_warn_deprecated_script_without_language",
			"Scripts without a defined language currently fall back to using the default shell for your platform. This fallback mechanic will soon stop working and a language will need to be explicitly defined for each script. Please configure the '[ACTIONABLE]language[/RESET]' field with a valid option (one of [ACTIONABLE]{{.V0}}[/RESET])",
			strings.Join(language.RecognizedNames(), ", ")))
	}

	out.Notice(output.Heading(locale.Tl("script_output", "Script Output")))
	return scriptrunner.Run(script, args)
}
