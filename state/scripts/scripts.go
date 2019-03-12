package scripts

import (
	"fmt"

	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/spf13/cobra"
)

// Command holds the definition for "state scripts".
var Command *commands.Command

func init() {
	Command = &commands.Command{
		Name:               "scripts",
		Description:        "scripts_description",
		Run:                Execute,
		DisableFlagParsing: true,
	}
}

// Execute the scripts command.
func Execute(cmd *cobra.Command, allArgs []string) {
	logging.Debug("Execute")
	prj := project.Get()
	scripts := prj.Scripts()
	for _, script := range scripts {
		fmt.Printf(" * %s\n", script.Name())
	}
}
