package keypair

import (
	"github.com/ActiveState/cli/internal/logging"
	"github.com/spf13/cobra"
)

// ExecuteRecipe processes the `export recipe` command.
func ExecuteRecipe(cmd *cobra.Command, args []string) {
	logging.Debug("Execute")
	cmd.Usage()
}
