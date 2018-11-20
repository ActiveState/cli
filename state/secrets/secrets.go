package secrets

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/api"
	"github.com/ActiveState/cli/internal/api/models"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	secretsapi "github.com/ActiveState/cli/internal/secrets-api"
	"github.com/ActiveState/cli/internal/secrets-api/client/secrets"
	secretsModels "github.com/ActiveState/cli/internal/secrets-api/models"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/ActiveState/cli/state/keypair"
	"github.com/ActiveState/cli/state/organizations"
	"github.com/ActiveState/cli/state/projects"
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
		SecretName  string
		SecretValue string
	}
}

// NewCommand creates a new Keypair command.
func NewCommand(secretsClient *secretsapi.Client) *Command {
	cmd := &Command{
		secretsClient: secretsClient,
	}

	cmd.config = &commands.Command{
		Name:        "secrets",
		Description: "secrets_cmd_description",
		Run:         cmd.Execute,
	}
	cmd.config.Append(buildSubcommandSet(cmd))

	return cmd
}

func buildSubcommandSet(cmd *Command) *commands.Command {
	return &commands.Command{
		Name:        "set",
		Description: "secrets_set_cmd_description",
		Run:         cmd.ExecuteSet,

		Flags: []*commands.Flag{
			&commands.Flag{
				Name:        "project",
				Shorthand:   "p",
				Description: "secrets_set_flag_project",
				Type:        commands.TypeBool,
				BoolVar:     &cmd.Flags.IsProject,
			},
			&commands.Flag{
				Name:        "user",
				Shorthand:   "u",
				Description: "secrets_set_flag_user",
				Type:        commands.TypeBool,
				BoolVar:     &cmd.Flags.IsUser,
			},
		},

		Arguments: []*commands.Argument{
			buildArgSecretName(cmd, true),
			buildArgSecretValue(cmd, true),
		},
	}
}

func buildArgSecretName(cmd *Command, required bool) *commands.Argument {
	return &commands.Argument{
		Name:        "secrets_arg_name_name",
		Description: "secrets_arg_name_description",
		Variable:    &cmd.Args.SecretName,
		Required:    required,
	}
}

func buildArgSecretValue(cmd *Command, required bool) *commands.Argument {
	return &commands.Argument{
		Name:        "secrets_arg_value_name",
		Description: "secrets_arg_value_description",
		Variable:    &cmd.Args.SecretValue,
		Required:    required,
	}
}

// Config returns the underlying commands.Command definition.
func (cmd *Command) Config() *commands.Command {
	return cmd.config
}

// Execute processes the secrets command.
func (cmd *Command) Execute(_ *cobra.Command, args []string) {
	project := projectfile.Get()
	org, failure := organizations.FetchByURLName(project.Owner)
	if failure == nil {
		failure = ListAll(cmd.secretsClient, org)
	}

	if failure != nil {
		failures.Handle(failure, locale.T("secrets_err"))
	}
}

// FetchAll fetchs the current user's secrets for an organization.
func FetchAll(secretsClient *secretsapi.Client, org *models.Organization) ([]*secretsModels.UserSecret, *failures.Failure) {
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

// ListAll prints a list of all of the UserSecrets names and their level for this user given an Organization.
func ListAll(secretsClient *secretsapi.Client, org *models.Organization) *failures.Failure {
	logging.Debug("listing user-secrets for org=%s", org.OrganizationID.String())

	orgProjects, failure := projects.FetchOrganizationProjects(org)
	if failure != nil {
		return failure
	}
	orgProjectMap := mapProjects(orgProjects)

	userSecrets, failure := FetchAll(secretsClient, org)
	if failure != nil {
		return failure
	} else if len(userSecrets) == 0 {
		return secretsapi.FailNotFound.New("secrets_err_no_secrets_found")
	}

	rows := [][]interface{}{}
	for _, userSecret := range userSecrets {
		rows = append(rows, []interface{}{*userSecret.Name, secretScopeDescription(userSecret, orgProjectMap)})
	}

	t := gotabulate.Create(rows)
	t.SetHeaders([]string{locale.T("secrets_col_name"), locale.T("secrets_col_scope")})
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
		return fmt.Sprintf("%s (%s)", locale.T("secrets_scope_user_project"), projName)
	} else if *userSecret.IsUser {
		return locale.T("secrets_scope_user_org")
	} else if userSecret.ProjectID != "" {
		return fmt.Sprintf("%s (%s)", locale.T("secrets_scope_project"), projName)
	} else {
		return locale.T("secrets_scope_org")
	}
}

// ExecuteSet processes the `secrets set` command.
func (cmd *Command) ExecuteSet(_ *cobra.Command, args []string) {
	projectFile := projectfile.Get()
	org, failure := organizations.FetchByURLName(projectFile.Owner)
	if failure == nil {
		var project *models.Project
		if cmd.Flags.IsProject {
			project, failure = projects.FetchByName(org, projectFile.Name)
		}

		if failure == nil {
			failure = UpsertUserSecret(cmd.secretsClient, org, project, cmd.Flags.IsUser, cmd.Args.SecretName, cmd.Args.SecretValue)
		}
	}

	if failure != nil {
		failures.Handle(failure, locale.T("secrets_err"))
	}
}

func findSecretByScope(userSecrets []*secretsModels.UserSecret, project *models.Project, isUser bool, secretName string) *secretsModels.UserSecret {
	// assuming we only have secrets for the correct organization, so we don't provide it as an arg
	// NOTE maybe just make a Secrets Service endpoint for this, eh?
	var projectIDStr string
	if project != nil {
		projectIDStr = project.ProjectID.String()
	}
	for _, userSecret := range userSecrets {
		if projectIDStr == userSecret.ProjectID.String() && *userSecret.IsUser == isUser && strings.EqualFold(*userSecret.Name, secretName) {
			return userSecret
		}
	}
	return nil
}

// UpsertUserSecret will add a new secret for this user or update an existing one. The update is dependent on
// the org, project, level, and name being the same as an existing secret.
func UpsertUserSecret(secretsClient *secretsapi.Client, org *models.Organization, project *models.Project, isUser bool, secretName, secretValue string) *failures.Failure {
	logging.Debug("attempting to upsert user-secret for org=%s", org.OrganizationID.String())
	kpOk, failure := keypair.Fetch(secretsClient)
	if failure != nil {
		return failure
	}

	kp, err := keypairs.ParseRSA(*kpOk.EncryptedPrivateKey)
	if err != nil {
		logging.Error("parsing user keypair: %v", err)
		return secretsapi.FailSave.New("keypair_err_parsing")
	}

	userSecrets, failure := FetchAll(secretsClient, org)
	if failure != nil {
		return failure
	}

	userSecret := findSecretByScope(userSecrets, project, isUser, secretName)

	encrBytes, err := kp.Encrypt([]byte(secretValue))
	if err != nil {
		logging.Error("encrypting user secret: %v", err)
		return secretsapi.FailSave.New("secrets_err_encrypting")
	}
	encrStr := base64.StdEncoding.EncodeToString(encrBytes)

	params := secrets.NewSaveAllUserSecretsParams()
	params.OrganizationID = org.OrganizationID
	secretChange := &secretsModels.UserSecretChange{
		Value: &encrStr,
	}
	if userSecret != nil {
		logging.Debug("updating UserSecret=%s", *userSecret.SecretID)
		secretChange.SecretID = *userSecret.SecretID
	} else {
		logging.Debug("adding UserSecret")
		secretChange.Name = secretName
		secretChange.IsUser = isUser
		if project != nil {
			secretChange.ProjectID = project.ProjectID
		}
	}

	params.UserSecrets = append(params.UserSecrets, secretChange)

	_, err = secretsClient.Secrets.Secrets.SaveAllUserSecrets(params, secretsClient.Auth)
	if err != nil {
		logging.Debug("error saving user secret: %v", err)
		return secretsapi.FailSave.New("secrets_err_save")
	}

	return nil
}
