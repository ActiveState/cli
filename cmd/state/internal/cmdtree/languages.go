package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/languages"
	"github.com/ActiveState/cli/pkg/project"
)

func newLanguagesCommand(prime *primer.Values) *captain.Command {
	runner := languages.NewLanguages(prime)

	return captain.NewCommand(
		"languages",
		locale.Tl("languages_title", "Listing Languages"),
		locale.T("languages_cmd_description"),
		prime.Output(),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, _ []string) error {
			return runner.Run()
		},
	).SetGroup(PlatformGroup)
}

func newLanguageUpdateCommand(prime *primer.Values) *captain.Command {
	runner := languages.NewUpdate(prime)

	params := languages.UpdateParams{}

	return captain.NewCommand(
		"update",
		locale.Tl("languages_update_title", "Updating Languages"),
		locale.T("languages_update_cmd_description"),
		prime.Output(),
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        "language",
				Description: locale.T("arg_languages_update_description"),
				Required:    true,
				Value:       &params.Language,
			},
		},
		func(ccmd *captain.Command, _ []string) error {
			proj, fail := project.GetSafe()
			if fail != nil {
				return fail
			}

			params.Owner = proj.Owner()
			params.ProjectName = proj.Name()
			return runner.Run(&params)
		},
	)
}
