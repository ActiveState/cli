package add

import (
	"fmt"

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

// Flags hold the flag values passed through the command line
var Flags struct {
	Constraint projectfile.Constraint
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

func parseConstraint(constraint string) projectfile.Constraint {
	// idx := strings.Index(constraint, "=")
	// key := constraint[:idx]
	// value := constraint[idx+1:]
	return projectfile.Constraint{"This isn't ", "done yet."}
}

// Execute the hook add command
// Adds a command to be run on the given hook trigger
func Execute(cmd *cobra.Command, args []string) {
	// Parse hook and statement from args
	logging.Debug("what is this crap? %v", args)
	hookname := args[0]
	command := args[1]
	// TODO I think --constraint could be passed multiple times
	// eg. --contrain platform=linux --constraint environment=blah
	// so likely will need to handle more than one contraint flag
	constraintFlag := cmd.LocalFlags().Lookup("constraint")
	// var constraint projectfile.Constraint
	if constraintFlag != nil {
		logging.Debug("******* Constraints not supported yet")
		//constraint = parseConstraint(constraintFlag.Value)
	}

	// Add hook to activestate.yaml for the active project
	project, err := projectfile.Get()
	if err != nil {
		logging.Error(fmt.Sprintf("Cannot add hook: %v", err))
		return
	}
	newHook := projectfile.Hook{Name: hookname, Value: command}
	project.Hooks = append(project.Hooks, newHook)

	logging.Debug("writing to config file")
	projectfile.Write(projectfile.GetProjectFilePath(), project)
	// with given statement and hash.
	logging.Debug("Execute `hook add`")
}
