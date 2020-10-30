package run

import (
	"github.com/ActiveState/cli/internal/failures"
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
	if !scriptrun.ProjectHasScript(proj, name) {
		fail := FailScriptNotDefined.New(
			locale.T("error_state_run_unknown_name", map[string]string{"Name": name}),
		)
		return fail
	}

	out.Notice(txtstyle.NewTitle(locale.Tl("run_script_title", "Running Script: [ACTIONABLE]{{.V0}}[/RESET]", name)))

	if authentication.Get().Authenticated() {
		checker.RunCommitsBehindNotifier(out)
	}

	return scriptrun.RunScript(out, subs, proj, name, args)
}
