package keypair

import (
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	secretsapi "github.com/ActiveState/cli/internal/secrets-api"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/spf13/cobra"
)

var (
	// FailKeypairParse identifies a failure during keypair parsing.
	FailKeypairParse = failures.Type("keypair.fail.parse", failures.FailUser)

	// FailInputPassphrase identifies a failure entering passphrase.
	FailInputPassphrase = failures.Type("keypair.fail.input.passphrase", failures.FailUserInput)
)

// Flags captures values for any of the flags used with the keypair command or its sub-commands.
var Flags struct {
	Bits           int
	DryRun         bool
	SkipPassphrase bool
}

// Command holds the definition for the `keypair` command.
var Command = &commands.Command{
	Name:        "keypair",
	Description: "keypair_cmd_description",
	Run:         Execute,
}

// GenerateCommand is a sub-command of keypair.
var GenerateCommand = &commands.Command{
	Name:        "generate",
	Description: "keypair_generate_cmd_description",
	Run:         ExecuteGenerate,
	Flags: []*commands.Flag{
		&commands.Flag{
			Name:        "bits",
			Shorthand:   "b",
			Description: "keypair_generate_flag_bits",
			Type:        commands.TypeInt,
			IntVar:      &Flags.Bits,
			IntValue:    constants.DefaultRSABitLength,
		},
		&commands.Flag{
			Name:        "dry-run",
			Shorthand:   "",
			Description: "keypair_generate_flag_dryrun",
			Type:        commands.TypeBool,
			BoolVar:     &Flags.DryRun,
		},
		&commands.Flag{
			Name:        "skip-passphrase",
			Shorthand:   "",
			Description: "keypair_generate_flag_skippassphrase",
			Type:        commands.TypeBool,
			BoolVar:     &Flags.SkipPassphrase,
		},
	},
}

// AuthCommand is a sub-command of keypair.
var AuthCommand = &commands.Command{
	Name:        "auth",
	Description: "keypair_auth_cmd_description",
	Run:         ExecuteAuth,
}

func init() {
	Command.Append(AuthCommand)
	Command.Append(GenerateCommand)
}

// Execute processes the keypair command.
func Execute(_ *cobra.Command, args []string) {
	secretsClient := secretsapi.DefaultClient
	uid, failure := secretsClient.AuthenticatedUserID()

	if failure == nil {
		logging.Debug("(secrets.keypair) authenticated user=%s", uid.String())
		failure = printEncodedKeypair(secretsClient)
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
