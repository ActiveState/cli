package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runners/push"
	"github.com/spf13/viper"
)

func newPushCommand() *captain.Command {
	pushRunner := push.NewPush(viper.GetViper())

	return captain.NewCommand(
		"push",
		locale.T("push_description"),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, args []string) error {
			return pushRunner.Run()
		},
	)
}
