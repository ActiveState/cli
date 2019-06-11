package secrets

import (
	"fmt"
	"os"
	"strings"

	"github.com/bndr/gotabulate"
	"github.com/spf13/cobra"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/secrets"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
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
			Name:        "secrets",
			Aliases:     []string{"variables", "vars"},
			Description: "secrets_cmd_description",
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
	if strings.HasPrefix(os.Args[1], "var") {
		print.Warning(locale.T("secrets_warn_deprecated_var"))
	}

	rows, failure := cmd.secretRows()
	if failure != nil {
		failures.Handle(failure, locale.T("secrets_err"))
	}

	t := gotabulate.Create(rows)
	t.SetHeaders([]string{locale.T("secrets_header_name"), locale.T("secrets_header_scope"), locale.T("secrets_header_usage")})
	t.SetHideLines([]string{"betweenLine", "top", "aboveTitle", "LineTop", "LineBottom", "bottomLine"}) // Don't print whitespace lines
	t.SetAlign("left")
	print.Line(t.Render("simple"))
}

// secretRows returns the rows used in our output table
func (cmd *Command) secretRows() ([][]interface{}, *failures.Failure) {
	prj := projectfile.Get()
	logging.Debug("listing variables for org=%s, project=%s", prj.Owner, prj.Name)

	secrets, fail := secrets.ByProject(cmd.secretsClient, projectfile.Get())
	if fail != nil {
		return nil, fail
	}

	rows := [][]interface{}{}
	for _, secret := range secrets {
		scope := string(project.SecretScopeProject)
		if secret.IsUser != nil && *secret.IsUser {
			scope = string(project.SecretScopeUser)
		}
		rows = append(rows, []interface{}{*secret.Name, scope, fmt.Sprintf("%s.%s", scope, *secret.Name)})
	}
	return rows, nil
}
