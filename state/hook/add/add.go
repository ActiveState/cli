package add

import (
	"fmt"

	"github.com/ActiveState/cli/internal/failures"
	funk "github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/cmdlets/hooks"
	"github.com/ActiveState/cli/pkg/projectfile"

	"github.com/ActiveState/cli/internal/logging"
	"github.com/spf13/cobra"
)

// KnownHooks records all known hooks
var KnownHooks = []string{"ACTIVATE"}

// Args hold the arg values passed through the command line
var Args struct {
	Hook   string
	Script string
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
					return failures.FailUserInput.New("error_hook_add_invalid_hook", value)
				}
				return nil
			},
		},
		&commands.Argument{
			Name:        "arg_hook_add_script",
			Description: "arg_hook_add_script_description",
			Variable:    &Args.Script,
			Required:    true,
		},
	},
}

// Execute the hook add command
// Adds a command to be run on the given hook trigger
func Execute(cmd *cobra.Command, args []string) {
	// Add hook to activestate.yaml for the active project
	project := projectfile.Get()

	newHook := projectfile.Hook{Name: Args.Hook, Value: Args.Script}

	exists, err := hooks.HookExists(newHook, project)
	if err != nil {
		failures.Handle(err, locale.T("hook_add_cannot_add_hook", Args))
		return
	}
	if exists {
		fmt.Printf(locale.T("hook_add_cannot_add_existing_hook"))
		return
	}
	project.Hooks = append(project.Hooks, newHook)
	project.Save()
	logging.Debug("Execute `hook add`")
}
