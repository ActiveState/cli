package cmdtree

import (
	"os"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runners/clean"
)

func newCleanCommand(outputer output.Outputer) *captain.Command {
	return captain.NewCommand(
		"clean",
		locale.T("clean_description"),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, _ []string) error {
			outputer.Print(ccmd.Help())
			return nil
		},
	)
}

func newUninstallCommand(outputer output.Outputer) *captain.Command {
	runner := clean.NewUninstall(outputer, prompt.New())
	params := clean.UninstallParams{}
	return captain.NewCommand(
		"uninstall",
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
			installPath, err := os.Executable()
			if err != nil {
				return err
			}

			params.ConfigPath = config.ConfigPath()
			params.CachePath = config.CachePath()
			params.InstallPath = installPath

			return runner.Run(&params)
		},
	)
}

func newCacheCommand(output output.Outputer) *captain.Command {
	runner := clean.NewCache(output, prompt.New())
	params := clean.CacheParams{}
	return captain.NewCommand(
		"cache",
		locale.T("cache_description"),
		[]*captain.Flag{
			{
				Name:        "force",
				Shorthand:   "f",
				Description: locale.T("flag_state_clean_cache_force_description"),
				Value:       &params.Force,
			},
		},
		[]*captain.Argument{},
		func(ccmd *captain.Command, _ []string) error {
			params.Path = config.CachePath()
			return runner.Run(&params)
		},
	)
}

func newConfigCommand(output output.Outputer) *captain.Command {
	runner := clean.NewConfig(output, prompt.New())
	params := clean.ConfigParams{}
	return captain.NewCommand(
		"config",
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
			params.Path = config.ConfigPath()
			return runner.Run(&params)
		},
	)
}
