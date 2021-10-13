package cmdtree

import (
	"strings"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/secrets"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
)

func newSecretsCommand(secretsClient *secretsapi.Client, prime *primer.Values) *captain.Command {
	runner := secrets.NewList(secretsClient, prime)

	params := secrets.ListRunParams{}

	ccmd := captain.NewCommand(
		"secrets",
		locale.Tl("secrets_title", "Secrets"),
		locale.T("secrets_cmd_description"),
		prime,
		[]*captain.Flag{
			{
				Name:        "filter-usedby",
				Description: locale.T("secrets_flag_filter"),
				Value:       &params.Filter,
			},
		},
		nil,
		func(_ *captain.Command, args []string) error {
			if len(args) > 0 && strings.HasPrefix(args[0], "var") {
				prime.Output().Error(locale.T("secrets_warn_deprecated_var"))
			}

			return runner.Run(params)
		},
	)

	ccmd.SetGroup(PlatformGroup)

	ccmd.SetAliases("variables", "vars")

	return ccmd
}

func newSecretsGetCommand(prime *primer.Values) *captain.Command {
	runner := secrets.NewGet(prime)

	params := secrets.GetRunParams{}

	return captain.NewCommand(
		"get",
		locale.Tl("secrets_get_title", "Getting Secret"),
		locale.T("secrets_get_cmd_description"),
		prime,
		nil,
		[]*captain.Argument{
			{
				Name:        locale.T("secrets_get_arg_name_name"),
				Description: locale.T("secrets_get_arg_name_description"),
				Value:       &params.Name,
				Required:    true,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params)
		},
	)
}

func newSecretsSetCommand(prime *primer.Values) *captain.Command {
	runner := secrets.NewSet(prime)

	params := secrets.SetRunParams{}

	return captain.NewCommand(
		"set",
		locale.Tl("secrets_set_title", "Setting Secret"),
		locale.T("secrets_set_cmd_description"),
		prime,
		nil,
		[]*captain.Argument{
			{
				Name:        locale.T("secrets_set_arg_name_name"),
				Description: locale.T("secrets_set_arg_name_description"),
				Value:       &params.Name,
				Required:    true,
			},
			{
				Name:        locale.T("secrets_set_arg_value_name"),
				Description: locale.T("secrets_set_arg_value_description"),
				Value:       &params.Value,
				Required:    true,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params)
		},
	)
}

func newSecretsSyncCommand(secretsClient *secretsapi.Client, prime *primer.Values) *captain.Command {
	runner := secrets.NewSync(secretsClient, prime)

	return captain.NewCommand(
		"sync",
		locale.Tl("secrets_sync_title", "Synchronizing Secrets"),
		locale.T("secrets_sync_cmd_description"),
		prime,
		nil,
		nil,
		func(_ *captain.Command, _ []string) error {
			return runner.Run()
		},
	)
}
