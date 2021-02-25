package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/platforms"
)

func newPlatformsCommand(prime *primer.Values) *captain.Command {
	runner := platforms.NewList(prime)

	return captain.NewCommand(
		"platforms",
		locale.Tl("platforms_title", "Listing Platforms"),
		locale.T("platforms_cmd_description"),
		prime.Output(),
		prime.Config(),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(_ *captain.Command, _ []string) error {
			return runner.Run()
		},
	).SetGroup(PlatformGroup)
}

func newPlatformsSearchCommand(prime *primer.Values) *captain.Command {
	runner := platforms.NewSearch(prime)

	return captain.NewCommand(
		"search",
		locale.Tl("platforms_search_title", "Searching Platforms"),
		locale.T("platforms_search_cmd_description"),
		prime.Output(),
		prime.Config(),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(_ *captain.Command, _ []string) error {
			return runner.Run()
		},
	)
}

func newPlatformsAddCommand(prime *primer.Values) *captain.Command {
	runner := platforms.NewAdd(prime)

	params := platforms.AddRunParams{}

	return captain.NewCommand(
		"add",
		locale.Tl("platforms_add_title", "Adding Platform"),
		locale.T("platforms_add_cmd_description"),
		prime.Output(),
		prime.Config(),
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
				Value:       &params.Platform,
				Required:    true,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params)
		},
	)
}

func newPlatformsRemoveCommand(prime *primer.Values) *captain.Command {
	runner := platforms.NewRemove(prime)

	params := platforms.RemoveRunParams{}

	return captain.NewCommand(
		"remove",
		locale.Tl("platforms_remove_title", "Removing Platform"),
		locale.T("platforms_remove_cmd_description"),
		prime.Output(),
		prime.Config(),
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
				Value:       &params.Platform,
				Required:    true,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params)
		},
	)
}
