package install

import (
	"github.com/ActiveState/ActiveState-CLI/internal/locale"
	"github.com/ActiveState/cobra"
	"github.com/dvirsky/go-pylog/logging"
)

var T = locale.T

var installCmd *cobra.Command

func Execute(cmd *cobra.Command, args []string) {
	logging.Debug("Execute")
}

// Register the install command
func Register(command *cobra.Command) {
	logging.Debug("Register")

	installCmd = &cobra.Command{
		Use:   "install",
		Short: T("install_project"),
		Run:   Execute,
	}

	command.AddCommand(installCmd)
}
