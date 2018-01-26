package activate

import (
	"github.com/ActiveState/ActiveState-CLI/internal/structures"
	"github.com/ActiveState/cobra"
	"github.com/dvirsky/go-pylog/logging"
)

// Command holds our main command definition
var Command = &structures.Command{
	Name:        "activate",
	Description: "activate_project",
	Run:         Execute,
}

// Execute the activate command
func Execute(cmd *cobra.Command, args []string) {
	logging.Debug("Execute")
}
