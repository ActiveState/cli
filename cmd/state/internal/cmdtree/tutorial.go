package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/tutorial"
)

func newTutorialCommand(prime *primer.Values) *captain.Command {
	cmd := captain.NewCommand(
		"tutorial",
		locale.Tl("tutorial_description", "Learn how to use the State Tool"),
		nil,
		nil,
		func(ccmd *captain.Command, args []string) error {
			prime.Output().Print(ccmd.UsageText())
			return nil
		},
	)

	return cmd
}

func newTutorialProjectCommand(prime *primer.Values) *captain.Command {
	runner := tutorial.New(prime)
	params := tutorial.NewProjectParams{
		ShowIntro: true,
	}

	cmd := captain.NewCommand(
		"new-project",
		locale.Tl("tutorial_description", "Learn how to create new projects (ie. virtual environments)"),
		[]*captain.Flag{
			{
				Name:        "show-intro",
				Description: locale.Tl("arg_tutorial_showintro", "Show Introduction Text"),
				Value:       &params.ShowIntro,
			},
			{
				Name:        "language",
				Description: locale.Tl("arg_tutorial_language", "Language that this new project should use"),
				Value:       &params.Language,
			},
		},
		nil,
		func(ccmd *captain.Command, args []string) error {
			return runner.RunNewProject(params)
		},
	)

	return cmd
}
