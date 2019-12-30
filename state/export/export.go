package export

import (
	"github.com/spf13/cobra"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
)

// Command is the export command's definition.
var Command = &commands.Command{
	Name:        "export",
	Description: "export_cmd_description",
	Run:         Execute,
}

var Flags struct {
	Verbose *bool
}

func init() {
	Command.Append(RecipeCommand)
	Command.Append(JWTCommand)
	Command.Append(PrivKeyCommand)
}

// Execute the export command.
func Execute(cmd *cobra.Command, args []string) {
	logging.CurrentHandler().SetVerbose(*Flags.Verbose)
	logging.Debug("Execute")
	err := cmd.Help()
	if err != nil {
		failures.Handle(err, locale.T("package_err_help"))
	}
}
