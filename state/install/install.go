package install

import (
	"github.com/ActiveState/ActiveState-CLI/internal/structures"
	"github.com/ActiveState/cobra"
	"github.com/dvirsky/go-pylog/logging"
)

// Command holds our main command definition
var Command = &structures.Command{
	Name:        "install",
	Description: "install_project",
	Run:         Execute,
}

// Execute the install command
func Execute(cmd *cobra.Command, args []string) {
	logging.Debug("Execute")
}
