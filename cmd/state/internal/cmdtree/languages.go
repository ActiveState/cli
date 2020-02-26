package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runners/languages"
	"github.com/ActiveState/cli/pkg/project"
)

func newLanguagesCommand(outputer output.Outputer) *captain.Command {
	runner := languages.NewLanguages()

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

			params := languages.NewLanguagesParams(proj.Owner(), proj.Name(), outputer)
			return runner.Run(&params)
		},
	)
}
