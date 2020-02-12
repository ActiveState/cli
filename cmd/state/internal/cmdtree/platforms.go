package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runners/platforms"
	"github.com/ActiveState/cli/pkg/project"
)

func newPlatformsCommand(out output.Outputer) *captain.Command {
	runner := platforms.NewList(project.GetSafe, out)

	return captain.NewCommand(
		"platforms",
		locale.T("platforms_cmd_description"),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(_ *captain.Command, _ []string) error {
			return runner.Run()
		},
	)
}

func newPlatformsSearchCommand(out output.Outputer) *captain.Command {
	runner := platforms.NewSearch(out)

	return captain.NewCommand(
		"search",
		locale.T("platforms_search_cmd_description"),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(_ *captain.Command, _ []string) error {
			return runner.Run()
		},
	)
}

func newPlatformsAddCommand(out output.Outputer) *captain.Command {
	runner := platforms.NewAdd(project.GetSafe)

	params := platforms.RunAddParams{}

	return captain.NewCommand(
		"add",
		locale.T("platforms_add_cmd_description"),
		[]*captain.Flag{
			{
				Name:        locale.T("flag_platforms_shared_bitwidth"),
				Description: locale.T("flag_platforms_shared_bitwidth_description"),
				Value:       &params.BitWidth,
			},
		},
		[]*captain.Argument{
			{
				Name:        locale.T("arg_platforms_shared_name"),
				Description: locale.T("arg_platforms_shared_name_description"),
				Value:       &params.Name,
			},
			{
				Name:        locale.T("arg_platforms_shared_version"),
				Description: locale.T("arg_platforms_shared_version_description"),
				Value:       &params.Version,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params)
		},
	)
}

func newPlatformsRemoveCommand(out output.Outputer) *captain.Command {
	runner := platforms.NewRemove(project.GetSafe)

	params := platforms.RunRemoveParams{}

	return captain.NewCommand(
		"remove",
		locale.T("platforms_remove_cmd_description"),
		[]*captain.Flag{
			{
				Name:        locale.T("flag_platforms_shared_bitwidth"),
				Description: locale.T("flag_platforms_shared_bitwidth_description"),
				Value:       &params.BitWidth,
			},
		},
		[]*captain.Argument{
			{
				Name:        locale.T("arg_platforms_shared_name"),
				Description: locale.T("arg_platforms_shared_name_description"),
				Value:       &params.Name,
			},
			{
				Name:        locale.T("arg_platforms_shared_version"),
				Description: locale.T("arg_platforms_shared_version_description"),
				Value:       &params.Version,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params)
		},
	)
}
