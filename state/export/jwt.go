package export

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

// JWTCommand is a sub-command of export.
var JWTCommand = &commands.Command{
	Name:        "jwt",
	Description: "export_jwt_cmd_description",
	Run:         ExecuteJWT,
}

// ExecuteJWT processes the `export recipe` command.
func ExecuteJWT(cmd *cobra.Command, args []string) {
	logging.Debug("Execute")

	if !authentication.Get().Authenticated() {
		print.Error(locale.T("err_command_requires_auth"))
		os.Exit(1)
	}

	print.Line(authentication.Get().BearerToken())
}
