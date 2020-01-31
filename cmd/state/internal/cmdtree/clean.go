package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runners/clean"
)

type CleanOpts struct {
	Force bool
}

func newCleanCommand() *captain.Command {
	runner := clean.NewClean(prompt.New())

	opts := CleanOpts{}
	return captain.NewCommand(
		"clean",
		locale.T("clean_description"),
		[]*captain.Flag{
			{
				Name:        "force",
				Shorthand:   "",
				Description: locale.T("flag_state_clean_force_description"),
				Type:        captain.TypeBool,
				BoolVar:     &opts.Force,
			},
		},
		[]*captain.Argument{},
		func(ccmd *captain.Command, _ []string) error {
			return runner.Run(&clean.RunParams{Force: opts.Force})
		},
	)
}
