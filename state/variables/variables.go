package variables

import (
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	secretsapi "github.com/ActiveState/cli/internal/secrets-api"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/bndr/gotabulate"
	"github.com/spf13/cobra"
)

// Command represents the secrets command and its dependencies.
type Command struct {
	config        *commands.Command
	secretsClient *secretsapi.Client

	Flags struct {
		IsProject bool
		IsUser    bool
	}

	Args struct {
		SecretName      string
		SecretValue     string
		ShareUserHandle string
	}
}

// NewCommand creates a new Keypair command.
func NewCommand(secretsClient *secretsapi.Client) *Command {
	cmd := &Command{
		secretsClient: secretsClient,
	}

	cmd.config = &commands.Command{
		Name:        "variables",
		Description: "variables_cmd_description",
		Run:         cmd.Execute,
	}

	cmd.config.Append(buildGetCommand(cmd))
	cmd.config.Append(buildSetCommand(cmd))
	cmd.config.Append(buildShareCommand(cmd))
	cmd.config.Append(buildSyncCommand(cmd))

	return cmd
}

// Config returns the underlying commands.Command definition.
func (cmd *Command) Config() *commands.Command {
	return cmd.config
}

// Execute processes the secrets command.
func (cmd *Command) Execute(_ *cobra.Command, args []string) {
	failure := listAllVariables(cmd.secretsClient)
	if failure != nil {
		failures.Handle(failure, locale.T("variables_err"))
	}
}

// listAllVariables prints a list of all of the UserSecrets names and their level for this user given an Organization.
func listAllVariables(secretsClient *secretsapi.Client) *failures.Failure {
	prj := project.Get()
	logging.Debug("listing variables for org=%s, project=%s", prj.Owner(), prj.Name())

	rows := [][]interface{}{}
	vars := prj.Variables()
	for _, v := range vars {
		value := ""
		valueCheck := v.ValueOrNil()
		encrypted := "-"
		shared := "-"
		if v.IsSecret() {
			if valueCheck == nil {
				value = locale.T("variables_value_secret_undefined")
			} else {
				value = locale.T("variables_value_secret")
			}
			encrypted = locale.T("confirmation")
			shared = string(*v.SharedWith())
		} else {
			value = *valueCheck
		}
		rows = append(rows, []interface{}{v.Name(), sanitizeValue(value), encrypted, shared})
	}

	t := gotabulate.Create(rows)
	t.SetHeaders([]string{locale.T("variables_col_name"), locale.T("variables_col_value"), locale.T("variables_col_encrypted"), locale.T("variables_col_shared")})
	t.SetAlign("left")

	print.Line(t.Render("simple"))

	return nil

}

// sanitizeValue will reduce the string length to 100 characters or the first line of text
func sanitizeValue(v string) string {
	v = strings.TrimSpace(v)
	breakPos := strings.Index(v, "\n")

	if len(v) > 100 {
		v = fmt.Sprintf("%s [..]", v[0:100])
	}
	if breakPos != -1 {
		v = fmt.Sprintf("%s [..]", v[0:breakPos])
	}

	return v
}
