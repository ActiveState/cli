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

// Command represents the keypair command and its dependencies.
type Command struct {
	Flags struct {
		Generate bool
	}

	config        *commands.Command
	secretsClient *secretsapi.Client
	generatorFn   keypairs.GeneratorFunc
}

// NewCommand creates a new Keypair command.
func NewCommand(secretsClient *secretsapi.Client, generatorFn keypairs.GeneratorFunc) *Command {
	cmd := &Command{
		secretsClient: secretsClient,
		generatorFn:   generatorFn,
	}

	cmd.config = &commands.Command{
		Name:        "keypair",
		Description: "secrets_keypair_cmd_description",
		Run:         cmd.Execute,

		Flags: []*commands.Flag{
			&commands.Flag{
				Name:        "generate",
				Shorthand:   "",
				Description: "secrets_keypair_generate_flag_usage",
				Type:        commands.TypeBool,
				BoolVar:     &cmd.Flags.Generate,
			},
		},
	}

	return cmd
}

// NewRSACommand creates a new Keypair command which assumes use of an RSA keypair generator.
// Will return an error if one is returned trying to create the new RSA keypair generator.
func NewRSACommand(secretsClient *secretsapi.Client) (*Command, error) {
	genFn, err := keypairs.NewRSAGeneratorFunc(4196)
	if err != nil {
		return nil, err
	}
	return NewCommand(secretsClient, genFn), nil
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
		if cmd.Flags.Generate {
			failure = Generate(cmd.secretsClient, cmd.generatorFn)
		} else {
			failure = Dump(cmd.secretsClient)
		}
	}

	if failure != nil {
		failures.Handle(failure, locale.T("secrets_keypair_err"))
	}
}

// Dump prints the encoded key-pair for the currently authenticated user to stdout.
func Dump(secretsClient *secretsapi.Client) *failures.Failure {
	getOk, err := secretsClient.Keys.GetKeypair(nil, secretsClient.Auth)
	if err != nil {
		if secretsapi.ErrorCode(err) == 404 {
			return secretsapi.FailNotFound.New("secrets_keypair_err_not_found")
		}
		return api.FailUnknown.Wrap(err)
	}
	print.Line(*getOk.Payload.EncryptedPrivateKey)
	print.Line(*getOk.Payload.PublicKey)
	return nil
}

// Generate implements the behavior to generate a new Secrets key-pair on behalf of the user
// and store that back to the Secrets Service.
func Generate(secretsClient *secretsapi.Client, generatorFn keypairs.GeneratorFunc) *failures.Failure {
	keypair, err := generatorFn()
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
		return secretsapi.FailSave.New("secrets_keypair_err_save")
	}

	print.Line("Keypair generated successfully")
	return nil
}
