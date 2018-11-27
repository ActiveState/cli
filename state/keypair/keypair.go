package keypair

import (
	"github.com/ActiveState/cli/internal/api"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/secrets-api"
	"github.com/ActiveState/cli/internal/secrets-api/client/keys"
	"github.com/ActiveState/cli/internal/secrets-api/models"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/spf13/cobra"
)

// DefaultRSABitLength represents the default RSA bit-length that will be assumed when
// generating new Keypairs.
const DefaultRSABitLength int = 4096

// Command represents the keypair command and its dependencies.
type Command struct {
	config        *commands.Command
	secretsClient *secretsapi.Client
}

// NewCommand creates a new Keypair command.
func NewCommand(secretsClient *secretsapi.Client) *Command {
	cmd := &Command{
		secretsClient: secretsClient,
	}

	cmd.config = &commands.Command{
		Name:        "keypair",
		Description: "keypair_cmd_description",
		Run:         cmd.Execute,
	}

	cmd.config.Append(&commands.Command{
		Name:        "generate",
		Description: "keypair_generate_cmd_description",
		Run:         cmd.ExecuteGenerate,
	})

	return cmd
}

// Config returns the underlying commands.Command definition.
func (cmd *Command) Config() *commands.Command {
	return cmd.config
}

// Execute processes the keypair command.
func (cmd *Command) Execute(_ *cobra.Command, args []string) {
	uid, failure := cmd.secretsClient.Authenticated()

	if failure == nil {
		logging.Debug("(secrets.keypair) authenticated user=%s", uid.String())
		failure = Dump(cmd.secretsClient)
	}

	if failure != nil {
		failures.Handle(failure, locale.T("keypair_err"))
	}
}

// ExecuteGenerate processes the `keypair generate` sub-command.
func (cmd *Command) ExecuteGenerate(_ *cobra.Command, args []string) {
	uid, failure := cmd.secretsClient.Authenticated()

	if failure == nil {
		logging.Debug("(secrets.keypair) authenticated user=%s", uid.String())
		failure = Generate(cmd.secretsClient)
	}

	if failure != nil {
		failures.Handle(failure, locale.T("keypair_err"))
	}
}

// Fetch fetchs the current user's keypair or returns a failure.
func Fetch(secretsClient *secretsapi.Client) (*models.Keypair, *failures.Failure) {
	getOk, err := secretsClient.Keys.GetKeypair(nil, secretsClient.Auth)
	if err != nil {
		if api.ErrorCode(err) == 404 {
			return nil, secretsapi.FailNotFound.New("keypair_err_not_found")
		}
		return nil, api.FailUnknown.Wrap(err)
	}
	return getOk.Payload, nil
}

// Dump prints the encoded key-pair for the currently authenticated user to stdout.
func Dump(secretsClient *secretsapi.Client) *failures.Failure {
	kp, failure := Fetch(secretsClient)
	if failure != nil {
		return failure
	}
	print.Line(*kp.EncryptedPrivateKey)
	print.Line(*kp.PublicKey)
	return nil
}

// Generate implements the behavior to generate a new Secrets key-pair on behalf of the user
// and store that back to the Secrets Service.
func Generate(secretsClient *secretsapi.Client) *failures.Failure {
	keypair, err := keypairs.GenerateRSA(DefaultRSABitLength)
	if err != nil {
		return api.FailUnknown.Wrap(err)
	}

	encodedPrivateKey := keypair.EncodePrivateKey()
	encodedPublicKey, err := keypair.EncodePublicKey()
	if err != nil {
		return api.FailUnknown.Wrap(err)
	}

	params := keys.NewSaveKeypairParams().WithKeypair(&models.KeypairChange{
		EncryptedPrivateKey: &encodedPrivateKey,
		PublicKey:           &encodedPublicKey,
	})

	_, err = secretsClient.Keys.SaveKeypair(params, secretsClient.Auth)
	if err != nil {
		return secretsapi.FailSave.New("keypair_err_save")
	}

	print.Line("Keypair generated successfully")
	return nil
}
