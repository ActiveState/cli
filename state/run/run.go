package run

import (
	"strings"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/spf13/cobra"
)

// Command holds the definition for "state run".
var Command *commands.Command

func init() {
	Command = &commands.Command{
		Name:               "run",
		Description:        "run_description",
		Run:                Execute,
		DisableFlagParsing: true,

		Arguments: []*commands.Argument{
			&commands.Argument{
				Name:        "arg_state_run_name",
				Description: "arg_state_run_name_description",
				Variable:    &Args.Name,
			},
		},
	}
}

// Args hold the arg values passed through the command line.
var Args struct {
	Name string
}

// Execute the run command.
func Execute(cmd *cobra.Command, allArgs []string) {
	prj := project.Get()
	behindCount, fail := model.CommitsBehindLatest(prj.Owner(), prj.Name(), prj.CommitID())
	if fail != nil {
		failures.Handle(fail, locale.T("err_could_not_get_commit_behind_count"))
	}
	if behindCount > 0 {
		print.Info(locale.T("runtime_update_available", behindCount))
	}

	logging.Debug("Execute")

	if Args.Name == "" || strings.HasPrefix(Args.Name, "-") {
		failures.Handle(failures.FailUserInput.New("error_state_run_undefined_name"), "")
		return
	}

	scriptArgs := allArgs[1:]

	// Determine which project script to run based on the given script name.
	script := prj.ScriptByName(Args.Name)
	if script == nil {
		print.Error(locale.T("error_state_run_unknown_name", map[string]string{"Name": Args.Name}))
		return
	}

	// Activate the state if needed.
	if !script.Standalone() && !subshell.IsActivated() {
		print.Info(locale.T("info_state_run_activating_state"))
		venv := virtualenvironment.Init()
		venv.OnDownloadArtifacts(func() { print.Line(locale.T("downloading_artifacts")) })
		venv.OnInstallArtifacts(func() { print.Line(locale.T("installing_artifacts")) })
		var fail = venv.Activate()
		if fail != nil {
			logging.Errorf("Unable to activate state: %s", fail.Error())
			failures.Handle(fail, locale.T("error_state_run_activate"))
			return
		}
	}

	// Run the script.
	scriptBlock := project.Expand(script.Value())
	subs, err := subshell.Get()
	if err != nil {
		failures.Handle(err, locale.T("error_state_run_no_shell"))
		return
	}

	print.Info(locale.Tr("info_state_run_running", script.Name(), script.Source().Path()))
	code, err := subs.Run(scriptBlock, scriptArgs...)
	if err != nil || code != 0 {
		failures.Handle(err, locale.T("error_state_run_error"))
		Command.Exiter(code)
		return
	}
}
