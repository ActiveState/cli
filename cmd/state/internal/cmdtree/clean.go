package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/clean"
)

func newCleanCommand(prime *primer.Values) *captain.Command {
	return captain.NewCommand(
		"clean",
		locale.Tl("clean_title", "Cleaning Resources"),
		locale.T("clean_description"),
		prime.Output(),
		prime.Config(),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, _ []string) error {
			prime.Output().Print(ccmd.Help())
			return nil
		},
	).SetGroup(UtilsGroup)
}

func newCleanUninstallCommand(prime *primer.Values) *captain.Command {
	params := clean.UninstallParams{}
	return captain.NewCommand(
		"uninstall",
		locale.Tl("clean_uninstall_title", "Uninstalling"),
		locale.T("uninstall_description"),
		prime.Output(),
		prime.Config(),
		[]*captain.Flag{
			{
				Name:        "force",
				Shorthand:   "f",
				Description: locale.T("flag_state_clean_uninstall_force_description"),
				Value:       &params.Force,
			},
			{
				Name:        "ignore-errors",
				Description: locale.T("flag_state_clean_ignore_errors_description"),
				Value:       &params.IgnoreErrors,
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

func newCleanCacheCommand(prime *primer.Values) *captain.Command {
	runner := clean.NewCache(prime)
	params := clean.CacheParams{}
	return captain.NewCommand(
		"cache",
		locale.Tl("clean_cache_title", "Cleaning Cached Runtimes"),
		locale.T("cache_description"),
		prime.Output(),
		prime.Config(),
		[]*captain.Flag{
			{
				Name:        "force",
				Shorthand:   "f",
				Description: locale.T("flag_state_clean_cache_force_description"),
				Value:       &params.Force,
			},
			{
				Name:        "ignore-errors",
				Description: locale.T("flag_state_clean_ignore_errors_description"),
				Value:       &params.IgnoreErrors,
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

func newCleanConfigCommand(prime *primer.Values) *captain.Command {
	runner := clean.NewConfig(prime)
	params := clean.ConfigParams{}
	return captain.NewCommand(
		"config",
		locale.Tl("clean_config_title", "Cleaning Configuration"),
		locale.T("config_description"),
		prime.Output(),
		prime.Config(),
		[]*captain.Flag{
			{
				Name:        "force",
				Shorthand:   "f",
				Description: locale.T("flag_state_config_cache_force_description"),
				Value:       &params.Force,
			},
			{
				Name:        "ignore-errors",
				Description: locale.T("flag_state_clean_ignore_errors_description"),
				Value:       &params.IgnoreErrors,
			},
		},
		[]*captain.Argument{},
		func(ccmd *captain.Command, _ []string) error {
			return runner.Run(&params)
		},
	)
}
