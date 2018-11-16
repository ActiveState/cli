package secrets

import (
	secretsapi "github.com/ActiveState/cli/internal/secrets-api"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/state/secrets/keypair"
	"github.com/spf13/cobra"
)

// Command represents the secrets command and its dependencies.
type Command struct {
	config        *commands.Command
	secretsClient *secretsapi.Client
}

// NewCommand creates a new Keypair command.
func NewCommand(secretsClient *secretsapi.Client) (*Command, error) {
	cmd := &Command{
		secretsClient: secretsClient,
	}

	cmd.config = &commands.Command{
		Name:        "secrets",
		Description: "secrets_cmd_description",
		Run:         cmd.Execute,
	}

	keypairCmd, err := keypair.NewRSACommand(secretsClient)
	if err != nil {
		return nil, err
	}
	cmd.config.Append(keypairCmd.Config())

	return cmd, nil
}

// Config returns the underlying commands.Command definition.
func (cmd *Command) Config() *commands.Command {
	return cmd.config
}

// Execute processes the secrets command.
func (cmd *Command) Execute(_ *cobra.Command, args []string) {
}
