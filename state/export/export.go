package export

import (
	"github.com/spf13/cobra"

	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
)

// Flags captures values for any of the flags used with the export command or its sub-commands.
var Flags struct {
	Pretty bool
}

// Args captures values for any of the args used with the export command or its sub-commands.
var Args struct {
	CommitID string
}

// Command is the export command's definition.
var Command = &commands.Command{
	Name:        "export",
	Description: "export_cmd_description",
	Run:         Execute,
}

// Execute the export command.
func Execute(cmd *cobra.Command, args []string) {
	logging.Debug("Execute")
	cmd.Usage()
}

func init() {
	Command.Append(RecipeCommand)
}

// RecipeCommand is a sub-command of export.
var RecipeCommand = &commands.Command{
	Name:        "recipe",
	Description: "export_recipe_cmd_description",
	Run:         ExecuteRecipe,
	Arguments: []*commands.Argument{
		&commands.Argument{
			Name:        "export_recipe_cmd_commitid_arg",
			Description: "export_recipe_cmd_commitid_arg_description",
			Variable:    &Args.CommitID,
		},
	},
	Flags: []*commands.Flag{
		&commands.Flag{
			Name:        "pretty",
			Shorthand:   "p",
			Description: "export_recipe_flag_pretty",
			Type:        commands.TypeBool,
			BoolVar:     &Flags.Pretty,
		},
	},
}
