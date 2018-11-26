package keypair

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/secrets-api"
	"github.com/ActiveState/cli/internal/secrets-api/client/keys"
	secretModels "github.com/ActiveState/cli/internal/secrets-api/models"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/spf13/cobra"
)

// DefaultRSABitLength represents the default RSA bit-length that will be assumed when
// generating new Keypairs.
const DefaultRSABitLength int = 4096

// FailKeypairParse identifies a failure during keypair parsing.
var FailKeypairParse = failures.Type("keypair.fail.parse", failures.FailUser)

// Command represents the keypair command and its dependencies.
type Command struct {
	config        *commands.Command
	secretsClient *secretsapi.Client

	Flags struct {
		Bits   int
		DryRun bool
	}
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
		Flags: []*commands.Flag{
			&commands.Flag{
				Name:        "bits",
				Shorthand:   "b",
				Description: "keypair_generate_flag_bits",
				Type:        commands.TypeInt,
				IntVar:      &cmd.Flags.Bits,
				IntValue:    DefaultRSABitLength,
			},
			&commands.Flag{
				Name:        "dry-run",
				Shorthand:   "",
				Description: "keypair_generate_flag_dryrun",
				Type:        commands.TypeBool,
				BoolVar:     &cmd.Flags.DryRun,
			},
		},
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
		failure = doDump(cmd.secretsClient)
	}

	if failure != nil {
		failures.Handle(failure, locale.T("keypair_err"))
	}
}

// doDump prints the encoded key-pair for the currently authenticated user to stdout.
func doDump(secretsClient *secretsapi.Client) *failures.Failure {
	kp, failure := keypairs.FetchRaw(secretsClient)
	if failure != nil {
		return failure
	}
	print.Line(*kp.EncryptedPrivateKey)
	print.Line(*kp.PublicKey)
	return nil
}

// ExecuteGenerate processes the `keypair generate` sub-command.
func (cmd *Command) ExecuteGenerate(_ *cobra.Command, args []string) {
	var failure *failures.Failure
	if !cmd.Flags.DryRun {
		_, failure = cmd.secretsClient.Authenticated()
	}

	if failure == nil {
		failure = doGenerate(cmd.secretsClient, cmd.Flags.Bits, cmd.Flags.DryRun)
	}

	if failure != nil {
		failures.Handle(failure, locale.T("keypair_err"))
	}
}

// doGenerate implements the behavior to generate a new Secrets key-pair on behalf of the user
// and store that back to the Secrets Service.
func doGenerate(secretsClient *secretsapi.Client, bits int, dryRun bool) *failures.Failure {
	keypair, failure := keypairs.GenerateRSA(bits)
	if failure != nil {
		return failure
	}

	encodedPrivateKey := keypair.EncodePrivateKey()
	encodedPublicKey, failure := keypair.EncodePublicKey()
	if failure != nil {
		return failure
	}

	if !dryRun {
		params := keys.NewSaveKeypairParams().WithKeypair(&secretModels.KeypairChange{
			EncryptedPrivateKey: &encodedPrivateKey,
			PublicKey:           &encodedPublicKey,
		})

		if _, err := secretsClient.Keys.SaveKeypair(params, secretsClient.Auth); err != nil {
			return secretsapi.FailSave.New("keypair_err_save")
		}
		print.Line("Keypair generated successfully")
	} else {
		print.Line(encodedPrivateKey)
		print.Line(encodedPublicKey)
	}

	return nil
}
