package scripts

import (
	"fmt"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/spf13/cobra"
)

// Command holds the definition for "state scripts".
var Command = &commands.Command{
	Name:        "scripts",
	Description: "scripts_description",
	Run:         Execute,
}

// Execute the scripts command.
func Execute(cmd *cobra.Command, allArgs []string) {
	logging.Debug("Execute")
	scripts := project.Get().Scripts()

	if len(scripts) == 0 {
		fmt.Println(locale.T("scripts_no_scripts"))
	}

	for _, script := range scripts {
		fmt.Printf(" * %s\n", script.Name())
	}
}
