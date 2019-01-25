package variables

import (
	"fmt"

	"github.com/ActiveState/cli/internal/api"
	"github.com/ActiveState/cli/internal/api/models"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/organizations"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/projects"
	secretsapi "github.com/ActiveState/cli/internal/secrets-api"
	"github.com/ActiveState/cli/internal/secrets-api/client/secrets"
	secretsModels "github.com/ActiveState/cli/internal/secrets-api/models"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/bndr/gotabulate"
	"github.com/go-openapi/strfmt"
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
	project := project.Get()
	org, failure := organizations.FetchByURLName(project.Owner())
	if failure == nil {
		failure = listAllUserSecrets(cmd.secretsClient, org)
	}

	if failure != nil {
		failures.Handle(failure, locale.T("variables_err"))
	}
}

// fetchAll fetchs the current user's secrets for an organization.
func fetchAll(secretsClient *secretsapi.Client, org *models.Organization) ([]*secretsModels.UserSecret, *failures.Failure) {
	params := secrets.NewGetAllUserSecretsParams()
	params.OrganizationID = org.OrganizationID
	getOk, err := secretsClient.Secrets.Secrets.GetAllUserSecrets(params, secretsClient.Auth)
	if err != nil {
		switch statusCode := api.ErrorCode(err); statusCode {
		case 401:
			return nil, api.FailAuth.New("err_api_not_authenticated")
		default:
			return nil, api.FailUnknown.Wrap(err)
		}
	}
	return getOk.Payload, nil
}

// listAllUserSecrets prints a list of all of the UserSecrets names and their level for this user given an Organization.
func listAllUserSecrets(secretsClient *secretsapi.Client, org *models.Organization) *failures.Failure {
	logging.Debug("listing user-secrets for org=%s", org.OrganizationID.String())

	orgProjects, failure := projects.FetchOrganizationProjects(org.Urlname)
	if failure != nil {
		return failure
	}
	orgProjectMap := mapProjects(orgProjects)

	userSecrets, failure := fetchAll(secretsClient, org)
	if failure != nil {
		return failure
	} else if len(userSecrets) == 0 {
		return secretsapi.FailUserSecretNotFound.New("variables_err_no_variables_found")
	}

	rows := [][]interface{}{}
	for _, userSecret := range userSecrets {
		rows = append(rows, []interface{}{*userSecret.Name, secretScopeDescription(userSecret, orgProjectMap)})
	}

	t := gotabulate.Create(rows)
	t.SetHeaders([]string{locale.T("variables_col_name"), locale.T("variables_col_scope")})
	t.SetAlign("left")

	print.Line(t.Render("simple"))

	return nil
}

type projectIDMap map[strfmt.UUID]*models.Project

func mapProjects(projects []*models.Project) projectIDMap {
	mapping := projectIDMap{}
	for _, proj := range projects {
		mapping[proj.ProjectID] = proj
	}
	return mapping
}

func secretScopeDescription(userSecret *secretsModels.UserSecret, projMap projectIDMap) string {
	projName := locale.T("undefined")
	if proj, found := projMap[userSecret.ProjectID]; found {
		projName = proj.Name
	}

	if *userSecret.IsUser && userSecret.ProjectID != "" {
		return fmt.Sprintf("%s (%s)", locale.T("variables_scope_user_project"), projName)
	} else if *userSecret.IsUser {
		return locale.T("variables_scope_user_org")
	} else if userSecret.ProjectID != "" {
		return fmt.Sprintf("%s (%s)", locale.T("variables_scope_project"), projName)
	} else {
		return locale.T("variables_scope_org")
	}
}
