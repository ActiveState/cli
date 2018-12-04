package keypair

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/secrets-api"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/spf13/cobra"
)

// DefaultRSABitLength represents the default RSA bit-length that will be assumed when
// generating new Keypairs.
const DefaultRSABitLength int = 4096

var (
	// FailKeypairParse identifies a failure during keypair parsing.
	FailKeypairParse = failures.Type("keypair.fail.parse", failures.FailUser)

	// FailInputPassphrase identifies a failure entering passphrase.
	FailInputPassphrase = failures.Type("keypair.fail.input.passphrase", failures.FailUserInput)
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

	cmd.config.Append(&commands.Command{
		Name:        "auth",
		Description: "keypair_auth_cmd_description",
		Run:         cmd.ExecuteAuth,
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
