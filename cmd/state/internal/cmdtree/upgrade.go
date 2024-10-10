package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/upgrade"
)

func newUpgradeCommand(prime *primer.Values) *captain.Command {
	runner := upgrade.New(prime)

	params := upgrade.NewParams()

	cmd := captain.NewCommand(
		"upgrade",
		locale.Tl("upgrade_cmd_title", "Upgrading Project"),
		locale.Tl("upgrade_cmd_description", "Upgrade dependencies of a project"),
		prime,
		[]*captain.Flag{
			{
				Name:        "ts",
				Description: locale.T("flag_state_upgrade_ts_description"),
				Value:       &params.Timestamp,
			},
			{
				Name:        "expand",
				Description: locale.T("flag_state_upgrade_expand_description"),
				Value:       &params.Expand,
			},
		},
		[]*captain.Argument{},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params)
		},
	)

	cmd.SetGroup(PackagesGroup)
	cmd.SetSupportsStructuredOutput()
	cmd.SetUnstable(true)

	return cmd
}
