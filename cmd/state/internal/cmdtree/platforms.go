package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/platforms"
	"github.com/ActiveState/cli/pkg/project"
)

func newPlatformsCommand(registry *captain.Registry, prime *primer.Values) *captain.Command {
	runner := platforms.NewList(prime)

	params := platforms.ListRunParams{}

	return registry.NewCommand(
		"platforms",
		locale.Tl("platforms_title", "Listing Platforms"),
		locale.T("platforms_cmd_description"),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(_ *captain.Command, _ []string) error {
			proj, fail := project.GetSafe()
			if fail != nil {
				return fail
			}
			params.Project = proj

			return runner.Run(params)
		},
	)
}

func newPlatformsSearchCommand(registry *captain.Registry, prime *primer.Values) *captain.Command {
	runner := platforms.NewSearch(prime)

	return registry.NewCommand(
		"search",
		locale.Tl("platforms_search_title", "Searching Platforms"),
		locale.T("platforms_search_cmd_description"),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(_ *captain.Command, _ []string) error {
			return runner.Run()
		},
	)
}

func newPlatformsAddCommand(registry *captain.Registry, prime *primer.Values) *captain.Command {
	runner := platforms.NewAdd(prime)

	params := platforms.AddRunParams{}

	return registry.NewCommand(
		"add",
		locale.Tl("platforms_add_title", "Adding Platform"),
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
				Required:    true,
			},
			{
				Name:        locale.T("arg_platforms_shared_version"),
				Description: locale.T("arg_platforms_shared_version_description"),
				Value:       &params.Version,
				Required:    true,
			},
		},
		func(_ *captain.Command, _ []string) error {
			proj, fail := project.GetSafe()
			if fail != nil {
				return fail
			}
			params.Project = proj

			return runner.Run(params)
		},
	)
}

func newPlatformsRemoveCommand(registry *captain.Registry, prime *primer.Values) *captain.Command {
	runner := platforms.NewRemove()

	params := platforms.RemoveRunParams{}

	return registry.NewCommand(
		"remove",
		locale.Tl("platforms_remove_title", "Removing Platform"),
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
				Required:    true,
			},
			{
				Name:        locale.T("arg_platforms_shared_version"),
				Description: locale.T("arg_platforms_shared_version_description"),
				Value:       &params.Version,
				Required:    true,
			},
		},
		func(_ *captain.Command, _ []string) error {
			proj, fail := project.GetSafe()
			if fail != nil {
				return fail
			}
			params.Project = proj

			return runner.Run(params)
		},
	)
}
