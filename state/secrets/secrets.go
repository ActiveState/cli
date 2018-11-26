package secrets

import (
	"encoding/base64"
	"fmt"

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
	"github.com/ActiveState/cli/internal/variables"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/projectfile"
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
		Name:        "secrets",
		Description: "secrets_cmd_description",
		Run:         cmd.Execute,
	}

	cmd.config.Append(buildSetCommand(cmd))
	cmd.config.Append(buildShareCommand(cmd))

	return cmd
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

// EncryptAndEncode will use the provided Encrypter to encrypt the plaintext value then it will
// base64 encode that ciphertext.
func EncryptAndEncode(encrypter keypairs.Encrypter, value string) (string, *failures.Failure) {
	encrBytes, failure := encrypter.Encrypt([]byte(value))
	if failure != nil {
		return "", secretsapi.FailSave.New("secrets_err_encrypting", failure.Error())
	}
	return base64.StdEncoding.EncodeToString(encrBytes), nil
}

// DecodeAndDecrypt will first base64 decode the provided value then it will use the provided
// Decrypter to decrypt the resulting ciphertext.
func DecodeAndDecrypt(decrypter keypairs.Decrypter, value string) (string, *failures.Failure) {
	encrBytes, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return "", secretsapi.FailSave.New("secrets_err_base64_decoding")
	}

	decrBytes, failure := decrypter.Decrypt(encrBytes)
	if failure != nil {
		return "", variables.FailExpandVariable.New("secrets_err_decrypting", failure.Error())
	}
	return string(decrBytes), nil
}
