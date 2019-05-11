package variables

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/bndr/gotabulate"
	"github.com/spf13/cobra"
)

// Command represents the secrets command and its dependencies.
type Command struct {
	config        *commands.Command
	secretsClient *secretsapi.Client

	Args struct {
		Name            string
		Value           string
		ShareUserHandle string
	}
}

// NewCommand creates a new Keypair command.
func NewCommand(secretsClient *secretsapi.Client) *Command {
	c := Command{
		secretsClient: secretsClient,
		config: &commands.Command{
			Name:        "variables",
			Aliases:     []string{"vars"},
			Description: "variables_cmd_description",
		},
	}
	c.config.Run = c.Execute

	c.config.Append(buildGetCommand(&c))
	c.config.Append(buildSetCommand(&c))
	c.config.Append(buildSyncCommand(&c))

	return &c
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

// listAllVariables prints a list of all of the variables defined for this project.
func listAllVariables(secretsClient *secretsapi.Client) *failures.Failure {
	prj := project.Get()
	logging.Debug("listing variables for org=%s, project=%s", prj.Owner(), prj.Name())

	hdrs, rows, failure := variablesTable(prj.Variables())
	if failure != nil {
		return failure
	}
	t := gotabulate.Create(rows)
	t.SetHeaders(hdrs)
	t.SetAlign("left")

	print.Line(t.Render("simple"))
	return nil
}

func variablesTable(vars []*project.Variable) (hdrs []string, rows [][]string, f *failures.Failure) {
	for _, v := range vars {
		isSetStatus, failure := v.IsSetStatus()
		if failure != nil {
			return nil, nil, failure
		}

		row := []string{
			v.Name(),
			v.Description(),
			isSetStatus,
			v.IsEncryptedStatus(),
			emptyToDash(v.SharedWith().String()),
			v.StoreLocation(),
		}
		rows = append(rows, row)
	}

	hdrs = []string{
		locale.T("variables_col_name"),
		locale.T("variables_col_description"),
		locale.T("variables_col_setunset"),
		locale.T("variables_col_encrypted"),
		locale.T("variables_col_shared"),
		locale.T("variables_col_store"),
	}

	return hdrs, rows, nil
}

func emptyToDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
