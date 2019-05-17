package shim

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
)

var Command *commands.Command

func init() {
	Command = &commands.Command{
		Name:        "shim",
		Description: "shim_description",
		Run:         Execute,
		Exiter:      os.Exit,

		DisableFlagParsing: true,
	}
}

func Execute(cmd *cobra.Command, args []string) {
	print.Info(locale.Tr("shim_disclaimer", filepath.Base(args[0])))

	runCmd := exec.Command(args[0], args[1:]...)
	runCmd.Stdin, runCmd.Stdout, runCmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	runCmd.Run()

	Command.Exiter(osutils.CmdExitCode(runCmd))
}
