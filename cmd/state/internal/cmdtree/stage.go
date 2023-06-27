package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/stage"
)

func newStageCommand(prime *primer.Values) *captain.Command {
	runner := stage.New(prime)

	cmd := captain.NewCommand(
		"stage",
		locale.Tl("stage_title", "Staging Changes"),
		locale.Tl("stage_description", "Stage changes to the Build Script"),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(_ *captain.Command, _ []string) error {
			return runner.Run()
		},
	)

	cmd.SetGroup(EnvironmentSetupGroup)

	return cmd
}
