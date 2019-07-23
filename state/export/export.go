package export

import (
	"github.com/spf13/cobra"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
)

// Flags captures values for any of the flags used with the export command or its sub-commands.
var Flags struct {
	Pretty   bool
	Platform string
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

func init() {
	Command.Append(RecipeCommand)
}

// Execute the export command.
func Execute(cmd *cobra.Command, args []string) {
	logging.Debug("Execute")
	err := cmd.Help()
	if err != nil {
		failures.Handle(err, locale.T("package_err_help"))
	}
}
