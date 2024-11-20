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
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, _ []string) error {
			prime.Output().Print(ccmd.Help())
			return nil
		},
	).SetGroup(UtilsGroup).SetSupportsStructuredOutput()
}

func newCleanUninstallCommand(prime *primer.Values, globals *globalOptions) *captain.Command {
	params := clean.UninstallParams{}
	return captain.NewCommand(
		"uninstall",
		locale.Tl("clean_uninstall_title", "Uninstalling"),
		locale.T("uninstall_description"),
		prime,
		[]*captain.Flag{
			{
				Name:        "all",
				Shorthand:   "a",
				Description: locale.Tl("flag_state_clean_uninstall_all", "Also delete all associated config and cache files"),
				Value:       &params.All,
			},
			{
				// This option is only used by the Windows uninstall shortcut to ask the user if they wish
				// to delete everything or keep cache and config. The user is also asked to press Enter
				// after the uninstall process is scheduled so they may note the printed log file path.
				Name:        "prompt",
				Description: "Asks the user if everything should be deleted or to keep cache and config",
				Hidden:      true, // this is not a user-facing flag
				Value:       &params.Prompt,
			},
		},
		[]*captain.Argument{},
		func(ccmd *captain.Command, _ []string) error {
			runner, err := clean.NewUninstall(prime)
			if err != nil {
				return err
			}

			if globals.NonInteractive {
				prime.Prompt().SetInteractive(false)
			}
			if globals.Force {
				prime.Prompt().SetForce(true)
				params.Force = true
			}
			return runner.Run(&params)
		},
	)
}

func newCleanCacheCommand(prime *primer.Values, globals *globalOptions) *captain.Command {
	runner := clean.NewCache(prime)
	params := clean.CacheParams{}
	return captain.NewCommand(
		"cache",
		locale.Tl("clean_cache_title", "Cleaning Cached Runtimes"),
		locale.T("cache_description"),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        "org/project",
				Description: locale.T("arg_state_clean_cache_project_description"),
				Required:    false,
				Value:       &params.Project,
			},
		},
		func(ccmd *captain.Command, _ []string) error {
			if globals.NonInteractive {
				prime.Prompt().SetInteractive(false)
			}
			return runner.Run(&params)
		},
	)
}

func newCleanConfigCommand(prime *primer.Values, globals *globalOptions) *captain.Command {
	runner := clean.NewConfig(prime)
	params := clean.ConfigParams{}
	return captain.NewCommand(
		"config",
		locale.Tl("clean_config_title", "Cleaning Configuration"),
		locale.T("clean_config_description"),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, _ []string) error {
			if globals.Force {
				prime.Prompt().SetForce(true)
				params.Force = true
			}
			return runner.Run(&params)
		},
	)
}
