package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/clean"
)

func newCleanCommand(registry *captain.Registry, prime *primer.Values) *captain.Command {
	return registry.NewCommand(
		"clean",
		locale.Tl("clean_title", "Cleaning Resources"),
		locale.T("clean_description"),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, _ []string) error {
			prime.Output().Print(ccmd.Help())
			return nil
		},
	)
}

func newUninstallCommand(registry *captain.Registry, prime *primer.Values) *captain.Command {
	params := clean.UninstallParams{}
	return registry.NewCommand(
		"uninstall",
		locale.Tl("clean_uninstall_title", "Uninstalling"),
		locale.T("uninstall_description"),
		[]*captain.Flag{
			{
				Name:        "force",
				Shorthand:   "f",
				Description: locale.T("flag_state_clean_uninstall_force_description"),
				Value:       &params.Force,
			},
		},
		[]*captain.Argument{},
		func(ccmd *captain.Command, _ []string) error {
			runner, err := clean.NewUninstall(prime)
			if err != nil {
				return err
			}

			return runner.Run(&params)
		},
	)
}

func newCacheCommand(registry *captain.Registry, prime *primer.Values) *captain.Command {
	runner := clean.NewCache(prime)
	params := clean.CacheParams{}
	return registry.NewCommand(
		"cache",
		locale.Tl("clean_cache_title", "Cleaning Cached Runtimes"),
		locale.T("cache_description"),
		[]*captain.Flag{
			{
				Name:        "force",
				Shorthand:   "f",
				Description: locale.T("flag_state_clean_cache_force_description"),
				Value:       &params.Force,
			},
		},
		[]*captain.Argument{
			{
				Name:        "project",
				Description: locale.T("arg_state_clean_cache_project_description"),
				Required:    false,
				Value:       &params.Project,
			},
		},
		func(ccmd *captain.Command, _ []string) error {
			return runner.Run(&params)
		},
	)
}

func newConfigCommand(registry *captain.Registry, prime *primer.Values) *captain.Command {
	runner := clean.NewConfig(prime)
	params := clean.ConfigParams{}
	return registry.NewCommand(
		"config",
		locale.Tl("clean_config_title", "Cleaning Configuration"),
		locale.T("config_description"),
		[]*captain.Flag{
			{
				Name:        "force",
				Shorthand:   "f",
				Description: locale.T("flag_state_config_cache_force_description"),
				Value:       &params.Force,
			},
		},
		[]*captain.Argument{},
		func(ccmd *captain.Command, _ []string) error {
			return runner.Run(&params)
		},
	)
}
