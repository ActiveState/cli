package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runners/platforms"
)

func newPlatformsCommand() *captain.Command {
	cmd := newPlatformsListCommand()

	return cmd.As("platforms", locale.T("platforms_cmd_description"))
}

func newPlatformsListCommand() *captain.Command {
	runner := platforms.NewList()

	return captain.NewCommand(
		"list",
		locale.T("platforms_list_cmd_description"),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(_ *captain.Command, _ []string) error {
			return runner.Run()
		})
}

func newPlatformsAddCommand() *captain.Command {
	runner := platforms.NewAdd()

	return captain.NewCommand(
		"add",
		locale.T("platforms_add_cmd_description"),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(_ *captain.Command, _ []string) error {
			return runner.Run()
		})
}

func newPlatformsRemoveCommand() *captain.Command {
	runner := platforms.NewRemove()

	return captain.NewCommand(
		"remove",
		locale.T("platforms_remove_cmd_description"),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(_ *captain.Command, _ []string) error {
			return runner.Run()
		})
}
