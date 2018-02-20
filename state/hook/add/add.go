package add

import (
	"github.com/ActiveState/ActiveState-CLI/internal/failures"
	"github.com/ActiveState/ActiveState-CLI/internal/print"
	"github.com/thoas/go-funk"

	"github.com/ActiveState/ActiveState-CLI/internal/locale"
	"github.com/ActiveState/ActiveState-CLI/pkg/cmdlets/commands"
	"github.com/ActiveState/ActiveState-CLI/pkg/projectfile"

	"github.com/ActiveState/ActiveState-CLI/internal/logging"
	"github.com/ActiveState/cobra"
)

// KnownHooks records all known hooks
var KnownHooks = []string{"ACTIVATE"}

// Args hold the arg values passed through the command line
var Args struct {
	Hook    string
	Command string
}

// Command Add
var Command = &commands.Command{
	Name:        "add",
	Description: "hook_add_description",
	Run:         Execute,

	Arguments: []*commands.Argument{
		&commands.Argument{
			Name:        "arg_hook_add_hook",
			Description: "arg_hook_add_hook_description",
			Variable:    &Args.Hook,
			Required:    true,
			Validator: func(arg *commands.Argument, value string) error {
				if !funk.Contains(KnownHooks, value) {
					return failures.User.New(locale.T("error_hook_add_invalid_hook", map[string]interface{}{"name": value}))
				}
				return nil
			},
		},
		&commands.Argument{
			Name:        "arg_hook_add_command",
			Description: "arg_hook_add_command_description",
			Variable:    &Args.Command,
			Required:    true,
		},
	},
}

// Execute the hook add command
// Adds a command to be run on the given hook trigger
func Execute(cmd *cobra.Command, args []string) {
	// Add hook to activestate.yaml for the active project
	project, err := projectfile.Get()
	if err != nil {
		msg := locale.T("hook_add_cannot_add_hook", Args)
		print.Error(msg)
		print.Error(err.Error())
		return
	}
	newHook := projectfile.Hook{Name: Args.Hook, Value: Args.Command}
	project.Hooks = append(project.Hooks, newHook)

	projectfile.Write(projectfile.GetProjectFilePath(), project)
	logging.Debug("Execute `hook add`")
}
