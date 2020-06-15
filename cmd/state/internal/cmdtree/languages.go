package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runners/languages"
	"github.com/ActiveState/cli/pkg/project"
)

func newLanguagesCommand(outputer output.Outputer) *captain.Command {
	runner := languages.NewLanguages(outputer)

	return captain.NewCommand(
		"languages",
		locale.T("languages_cmd_description"),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, _ []string) error {
			proj, fail := project.GetSafe()
			if fail != nil {
				return fail
			}

			params := languages.NewLanguagesParams(proj.Owner(), proj.Name())
			return runner.Run(&params)
		},
	)
}

func newLanguageUpdateCommand(outputer output.Outputer) *captain.Command {
	runner := languages.NewUpdate(outputer)

	params := languages.UpdateParams{}

	return captain.NewCommand(
		"update",
		locale.T("languages_update_cmd_description"),
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
