package run

import (
	"strings"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/output/txtstyle"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/scriptrun"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/pkg/cmdlets/checker"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/project"
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
		return failures.FailUserInput.New("error_state_run_undefined_name").ToError()
	}

	out.Notice(txtstyle.NewTitle(locale.Tl("run_script_title", "Running Script: [ACTIONABLE]{{.V0}}[/RESET]", name)))

	if authentication.Get().Authenticated() {
		checker.RunCommitsBehindNotifier(out)
	}

	script := proj.ScriptByName(name)
	if script == nil {
		return locale.NewInputError("error_state_run_unknown_name", "Script does not exist: {{.V0}}", name)
	}

	scriptrunner := scriptrun.New(out, subs, proj)
	if !script.Standalone() && scriptrunner.NeedsActivation() {
		out.Notice(output.Heading(locale.Tl("notice", "Notice")))
		out.Notice(locale.T("info_state_run_activating_state"))
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
