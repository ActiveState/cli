package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runners/platforms"
)

func newPlatformsCommand(out output.Outputer) *captain.Command {
	cmd := newPlatformsListCommand(out)

	return cmd.As("platforms", locale.T("platforms_cmd_description"))
}

func newPlatformsListCommand(out output.Outputer) *captain.Command {
	runner := platforms.NewList()

	return captain.NewCommand(
		"list",
		locale.T("platforms_list_cmd_description"),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(_ *captain.Command, _ []string) error {
			results, err := runner.Run()
			if err != nil {
				out.Error(err)
				return err
			}
			out.Print(results)
			return nil
		},
	)
}

func newPlatformsSearchCommand(out output.Outputer) *captain.Command {
	runner := platforms.NewSearch()

	return captain.NewCommand(
		"search",
		locale.T("platforms_search_cmd_description"),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(_ *captain.Command, _ []string) error {
			results, err := runner.Run()
			if err != nil {
				out.Error(err)
				return err
			}
			out.Print(results)
			return nil
		},
	)
}

func newPlatformsAddCommand(out output.Outputer) *captain.Command {
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

func newPlatformsRemoveCommand(out output.Outputer) *captain.Command {
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
