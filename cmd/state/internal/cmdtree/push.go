package cmdtree

import (
	"github.com/spf13/viper"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/push"
)

func newPushCommand(prime *primer.Values) *captain.Command {
	pushRunner := push.NewPush(viper.GetViper(), prime)

	return captain.NewCommand(
		"push",
		locale.Tl("push_title", "Pushing Local Project"),
		locale.T("push_description"),
		prime.Output(),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, args []string) error {
			return pushRunner.Run()
		},
	)
}
