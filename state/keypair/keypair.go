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
	"github.com/ActiveState/cli/internal/surveyor"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/spf13/cobra"
	survey "gopkg.in/AlecAivazis/survey.v1"
)

// DefaultRSABitLength represents the default RSA bit-length that will be assumed when
// generating new Keypairs.
const DefaultRSABitLength int = 4096

var (
	// FailKeypairParse identifies a failure during keypair parsing.
	FailKeypairParse = failures.Type("keypair.fail.parse", failures.FailUser)

	// FailInputPassphrase identifies a failure entering passphrase.
	FailInputPassphrase = failures.Type("keypair.fail.parse", failures.FailUserInput)
)

// Command represents the keypair command and its dependencies.
type Command struct {
	config        *commands.Command
	secretsClient *secretsapi.Client

	Flags struct {
		Bits           int
		DryRun         bool
		SkipPassphrase bool
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
			&commands.Flag{
				Name:        "skip-passphrase",
				Shorthand:   "",
				Description: "keypair_generate_flag_skippassphrase",
				Type:        commands.TypeBool,
				BoolVar:     &cmd.Flags.SkipPassphrase,
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
		failure = printEncodedKeypair(cmd.secretsClient)
	}

	if failure != nil {
		failures.Handle(failure, locale.T("keypair_err"))
	}
}

// printEncodedKeypair prints the encoded key-pair for the currently authenticated user to stdout.
func printEncodedKeypair(secretsClient *secretsapi.Client) *failures.Failure {
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
	var passphrase string
	var failure *failures.Failure

	if cmd.Flags.SkipPassphrase {
		// for the moment, we do not want to record any unencrypted private-keys
		cmd.Flags.DryRun = true
	}

	if !cmd.Flags.DryRun {
		// ensure user is authenticated before bothering to generate keypair and ask for passphrase
		_, failure = cmd.secretsClient.Authenticated()
	}

	if failure == nil && !cmd.Flags.SkipPassphrase {
		passphrase, failure = promptForPassphrase()
	}

	if failure == nil {
		failure = generateKeypair(cmd.secretsClient, passphrase, cmd.Flags.Bits, cmd.Flags.DryRun)
	}

	if failure != nil {
		failures.Handle(failure, locale.T("keypair_err"))
	}
}

func promptForPassphrase() (string, *failures.Failure) {
	var passphrase string
	var prompt = &survey.Password{Message: locale.T("passphrase_prompt")}
	if err := survey.AskOne(prompt, &passphrase, surveyor.ValidateRequired); err != nil {
		return "", FailInputPassphrase.New("keypair_err_passphrase_prompt")
	}
	return passphrase, nil
}

// generateKeypair implements the behavior to generate a new Secrets key-pair on behalf of the user
// and store that back to the Secrets Service. If dry-run is enabled, a keypair will be generated
// and printed, but not stored anywhere (thus, not used).
func generateKeypair(secretsClient *secretsapi.Client, passphrase string, bits int, dryRun bool) *failures.Failure {
	keypair, failure := keypairs.GenerateRSA(bits)
	if failure != nil {
		return failure
	}

	var encodedPrivateKey string
	if passphrase == "" {
		encodedPrivateKey = keypair.EncodePrivateKey()
	} else {
		encodedPrivateKey, failure = keypair.EncryptAndEncodePrivateKey(passphrase)
		if failure != nil {
			return failure
		}
	}

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
			return secretsapi.FailKeypairSave.New("keypair_err_save")
		}
		print.Line("Keypair generated successfully")

		// save the keypair locally to avoid authenticating the keypair every time it's used
		if failure = keypairs.Save(keypair, "private"); failure != nil {
			return failure
		}
	} else {
		print.Line(encodedPrivateKey)
		print.Line(encodedPublicKey)
	}

	return nil
}
