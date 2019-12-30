package export

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

// PrivKeyCommand is a sub-command of export.
var PrivKeyCommand = &commands.Command{
	Name:        "private-key",
	Description: "export_privkey_cmd_description",
	Run:         ExecutePrivKey,
}

// ExecutePrivKey processes the `export recipe` command.
func ExecutePrivKey(cmd *cobra.Command, args []string) {
	logging.CurrentHandler().SetVerbose(*Flags.Verbose)
	logging.Debug("Execute")

	if !authentication.Get().Authenticated() {
		print.Error(locale.T("err_command_requires_auth"))
		os.Exit(1)
	}

	filepath := keypairs.LocalKeyFilename(constants.KeypairLocalFileName)
	contents, fail := fileutils.ReadFile(filepath)
	if fail != nil {
		failures.Handle(fail, locale.T("err_read_privkey"))
		os.Exit(1)
	}
	print.Line(string(contents))
}
