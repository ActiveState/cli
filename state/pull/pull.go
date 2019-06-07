package pull

import (
	"github.com/spf13/cobra"

	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
)

// Command is the pull command's definition.
var Command = &commands.Command{
	Name:        "pull",
	Description: "pull_latest",
	Run:         Execute,
}

// Execute the pull command.
func Execute(cmd *cobra.Command, args []string) {
	logging.Debug("Execute")

	// request latest commit id from api

	// compare latest with project data

	// update as.yaml
	print.Line("pull cmd")
}
