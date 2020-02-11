package cmdtree

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runners/clean"
)

type CleanOpts struct {
	Force bool
}

func newCleanCommand(outputer output.Outputer) *captain.Command {
	runner := clean.NewClean(outputer, prompt.New())

	opts := CleanOpts{}
	return captain.NewCommand(
		"clean",
		locale.T("clean_description"),
		[]*captain.Flag{
			{
				Name:        "force",
				Shorthand:   "f",
				Description: locale.T("flag_state_clean_force_description"),
				Value:       &opts.Force,
			},
		},
		[]*captain.Argument{},
		func(ccmd *captain.Command, _ []string) error {
			installPath, err := filepath.Abs(os.Args[0])
			if err != nil {
				return err
			}

			return runner.Run(&clean.RunParams{
				Force:       opts.Force,
				ConfigPath:  config.ConfigPath(),
				CachePath:   config.CachePath(),
				InstallPath: installPath,
			})
		},
	)
}
