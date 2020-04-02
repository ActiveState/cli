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

type UninstallOpts struct {
	Force bool
}

func newUninstallCommand(outputer output.Outputer) *captain.Command {
	runner := clean.NewUninstall(outputer, prompt.New())
	opts := UninstallOpts{}
	return captain.NewCommand(
		"uninstall",
		locale.T("uninstall_description"),
		[]*captain.Flag{
			{
				Name:        "force",
				Shorthand:   "f",
				Description: locale.T("flag_state_clean_uninstall_force_description"),
				Value:       &opts.Force,
			},
		},
		[]*captain.Argument{},
		func(ccmd *captain.Command, _ []string) error {
			installPath, err := os.Executable()
			if err != nil {
				return err
			}

			return runner.Run(&clean.UninstallParams{
				Force:       opts.Force,
				ConfigPath:  config.ConfigPath(),
				CachePath:   config.CachePath(),
				InstallPath: installPath,
			})
		},
	)
}

type CacheOpts struct {
	Force bool
}

func newCacheCommand(output output.Outputer) *captain.Command {
	runner := clean.NewCache(output, prompt.New())
	opts := CacheOpts{}
	return captain.NewCommand(
		"cache",
		locale.T("cache_description"),
		[]*captain.Flag{
			{
				Name:        "force",
				Shorthand:   "f",
				Description: locale.T("flag_state_clean_cache_force_description"),
				Value:       &opts.Force,
			},
		},
		[]*captain.Argument{},
		func(ccmd *captain.Command, _ []string) error {
			return runner.Run(&clean.CacheParams{
				Force: opts.Force,
				Path:  config.CachePath(),
			})
		},
	)
}

type ConfigOpts struct {
	Force bool
}

func newConfigCommand(output output.Outputer) *captain.Command {
	runner := clean.NewConfig(output, prompt.New())
	opts := ConfigOpts{}
	return captain.NewCommand(
		"config",
		locale.T("config_description"),
		[]*captain.Flag{
			{
				Name:        "force",
				Shorthand:   "f",
				Description: locale.T("flag_state_config_cache_force_description"),
				Value:       &opts.Force,
			},
		},
		[]*captain.Argument{},
		func(ccmd *captain.Command, _ []string) error {
			return runner.Run(&clean.ConfigParams{
				Force: opts.Force,
				Path:  config.ConfigPath(),
			})
		},
	)
}
