package export

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
)

// Command is the export command's definition.
var Command = &commands.Command{
	Name:        "export",
	Description: "export_cmd_description",
	Run:         Execute,
}

// Execute the pull command.
func Execute(cmd *cobra.Command, args []string) {
	logging.Debug("Execute")
	fmt.Println(cmd.Name())
	panic("TEST")
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
}
