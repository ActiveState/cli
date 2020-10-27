package cmdtree

import (
	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/tutorial"
)

func newTutorialCommand(prime *primer.Values) *captain.Command {
	cmd := captain.NewCommand(
		"tutorial",
		locale.Tl("tutorial_title", "Running Tutorial"),
		locale.Tl("tutorial_description", "Learn how to use the State Tool"),
		prime.Output(),
		nil,
		nil,
		func(ccmd *captain.Command, args []string) error {
			prime.Output().Print(ccmd.UsageText())
			return nil
		},
	)

	cmd.SetHidden(true)

	return cmd
}

func newTutorialProjectCommand(prime *primer.Values) *captain.Command {
	runner := tutorial.New(prime)
	params := tutorial.NewProjectParams{}

	cmd := captain.NewCommand(
		"new-project",
		locale.Tl("tutorial_new_project", `Running "New Project" Tutorial`),
		locale.Tl("tutorial_description", "Learn how to create new projects. (ie. virtual environments)"),
		prime.Output(),
		[]*captain.Flag{
			{
				Name:        "skip-intro",
				Description: locale.Tl("arg_tutorial_showintro", "Skip Introduction Text"),
				Value:       &params.SkipIntro,
			},
			{
				Name:        "language",
				Description: locale.Tl("arg_tutorial_language", "Language that this new project should use"),
				Value:       &params.Language,
			},
		},
		nil,
		func(ccmd *captain.Command, args []string) error {
			err := runner.RunNewProject(params)
			if err != nil {
				analytics.EventWithLabel(analytics.CatTutorial, "error", errs.Join(err, " :: ").Error())
			} else {
				analytics.Event(analytics.CatTutorial, "completed")
			}
			return err
		},
	)

	return cmd
}
