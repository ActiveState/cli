package variables

import (
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/organizations"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/projects"
	secretsapi "github.com/ActiveState/cli/internal/secrets-api"
	secretsModels "github.com/ActiveState/cli/internal/secrets-api/models"
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

// userSecretsCurrentProject returns secrets relevant only to the current project
func userSecretsCurrentProject(secretsClient *secretsapi.Client) ([]*secretsModels.UserSecret, *failures.Failure) {
	prj := project.Get()

	orgModel, failure := organizations.FetchByURLName(prj.Owner())
	if failure != nil {
		return nil, failure
	}

	projectModel, failure := projects.FetchByName(prj.Owner(), prj.Name())
	if failure != nil {
		return nil, failure
	}

	userSecrets, failure := secretsapi.FetchAll(secretsClient, orgModel)
	if failure != nil {
		return nil, failure
	} else if len(userSecrets) == 0 {
		return userSecrets, secretsapi.FailUserSecretNotFound.New("variables_err_no_variables_found")
	}

	userSecretsFiltered := []*secretsModels.UserSecret{}
	for _, userSecret := range userSecrets {
		if (userSecret.ProjectID != "" && userSecret.ProjectID != projectModel.ProjectID) ||
			(userSecret.OrganizationID != nil && *userSecret.OrganizationID != orgModel.OrganizationID) {
			continue
		}
		userSecretsFiltered = append(userSecretsFiltered, userSecret)
	}

	return userSecretsFiltered, nil
}

// listAllVariables prints a list of all of the UserSecrets names and their level for this user given an Organization.
func listAllVariables(secretsClient *secretsapi.Client) *failures.Failure {
	prj := project.Get()
	logging.Debug("listing variables for org=%s, project=%s", prj.Owner(), prj.Name())

	userSecrets, failure := userSecretsCurrentProject(secretsClient)
	if failure != nil {
		return failure
	}

	rows := [][]interface{}{}
	for _, userSecret := range userSecrets {
		rows = append(rows, []interface{}{*userSecret.Name, locale.T("variables_value_secret"), secretScopeDescription(userSecret)})
	}

	projectVars := prj.Variables()
	for _, projectVar := range projectVars {
		rows = append(rows, []interface{}{projectVar.Name(), sanitizeValue(projectVar.Value()), locale.T("variables_scope_local")})
	}

	t := gotabulate.Create(rows)
	t.SetHeaders([]string{locale.T("variables_col_name"), locale.T("variables_col_value"), locale.T("variables_col_scope")})
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
