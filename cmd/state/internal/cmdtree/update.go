package cmdtree

import (
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/update"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
)

func newUpdateCommand(prime *primer.Values) *captain.Command {
	runner := update.New(prime)
	params := update.Params{}

	cmd := captain.NewCommand(
		"update",
		locale.Tl("update_title", "Updating The State Tool"),
		locale.Tl("update_description", "Updates the State Tool to the latest available version"),
		prime.Output(),
		[]*captain.Flag{
			{
				Name: "lock",
				Description: locale.Tl(
					"flag_update_lock",
					"Lock the State Tool at the current version, this disables automatic updates. You can still force an update by manually running the update command."),
				Value: &params.Lock,
			},
			{
				Name: "force",
				Description: locale.Tl(
					"flag_update_force",
					"Automatically confirm that you would like to update the State Tool version that your project is locked to."),
				Value: &params.Force,
			},
		},
		[]*captain.Argument{},
		func(cmd *captain.Command, args []string) error {
			return runner.Run(&params)
		},
	)
	cmd.SetGroup(UtilsGroup)
	cmd.SetSkipChecks(true)
	return cmd
}
