package shim

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/ActiveState/cli/internal/executor"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/logging"
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
	exe := executor.New(args[0], args[1:]...)
	exe.OnStdin(func(input []byte) {
		logging.Debug("STDIN: %s", string(input))
	})
	exe.OnStdout(func(output []byte) {
		logging.Debug("STDOUT: %s", string(output))
	})
	exe.OnStderr(func(output []byte) {
		logging.Debug("STDERR: %s", string(output))
	})
	fail := exe.Run()
	if fail != nil {
		failures.Handle(fail, "Command failed")
	}
}
