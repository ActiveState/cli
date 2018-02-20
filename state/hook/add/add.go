package add

import (
	"github.com/ActiveState/ActiveState-CLI/internal/print"

	"github.com/ActiveState/ActiveState-CLI/internal/locale"
	"github.com/ActiveState/ActiveState-CLI/internal/structures"
	"github.com/ActiveState/ActiveState-CLI/pkg/projectfile"

	"github.com/ActiveState/ActiveState-CLI/internal/logging"
	"github.com/ActiveState/cobra"
)

// Command Add
var Command = &structures.Command{
	Name:        "add",
	Description: "hook_add_description",
	Run:         Execute,
}

func init() {
	logging.Debug("Add init")
	//Command.GetCobraCmd().ValidArgs = append(Command.GetCobraCmd().ValidArgs, "trigger", "command")
	Command.GetCobraCmd().Args = validateArgs
}

func validateArgs(cmd *cobra.Command, args []string) error {
	// TODO lists of known hooks, warn if hook passed isn't supported
	// err := cobra.OnlyValidArgs(cmd, args)
	// if err != nil {
	// 	return err
	// }
	err := cobra.MinimumNArgs(2)(cmd, args)
	if err != nil {
		return err
	}
	return nil
}

// Execute the hook add command
// Adds a command to be run on the given hook trigger
func Execute(cmd *cobra.Command, args []string) {
	// Parse hook and statement from args
	hookname := args[0]
	command := args[1]

	// Add hook to activestate.yaml for the active project
	project, err := projectfile.Get()
	if err != nil {
		msg := locale.T("hook_add_cannot_add_hook", map[string]interface{}{"Hookname": hookname, "Cmd": command})
		print.Error(msg)
		print.Error(err.Error())
		return
	}
	newHook := projectfile.Hook{Name: hookname, Value: command}
	project.Hooks = append(project.Hooks, newHook)

	projectfile.Write(projectfile.GetProjectFilePath(), project)
	logging.Debug("Execute `hook add`")
}
