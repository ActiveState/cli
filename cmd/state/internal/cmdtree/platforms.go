package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runners/platforms"
)

func newPlatformsCommand(printer platforms.Printer) *captain.Command {
	cmd := newPlatformsListCommand(printer)

	return cmd.As("platforms", locale.T("platforms_cmd_description"))
}

func newPlatformsListCommand(printer platforms.Printer) *captain.Command {
	runner := platforms.NewList(printer)

	return captain.NewCommand(
		"list",
		locale.T("platforms_list_cmd_description"),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(_ *captain.Command, _ []string) error {
			return runner.Run()
		})
}

func newPlatformsAddCommand(printer platforms.Printer) *captain.Command {
	runner := platforms.NewAdd(printer)

	return captain.NewCommand(
		"add",
		locale.T("platforms_add_cmd_description"),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(_ *captain.Command, _ []string) error {
			return runner.Run()
		})
}

func newPlatformsRemoveCommand(printer platforms.Printer) *captain.Command {
	runner := platforms.NewRemove(printer)

	return captain.NewCommand(
		"remove",
		locale.T("platforms_remove_cmd_description"),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(_ *captain.Command, _ []string) error {
			return runner.Run()
		})
}
