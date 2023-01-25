package cmdtree

import (
	"github.com/ActiveState/cli/internal-as/captain"
	"github.com/ActiveState/cli/internal-as/locale"
	"github.com/ActiveState/cli/internal-as/primer"
	"github.com/ActiveState/cli/internal/runners/languages"
)

func newLanguagesCommand(prime *primer.Values) *captain.Command {
	runner := languages.NewLanguages(prime)

	return captain.NewCommand(
		"languages",
		locale.Tl("languages_title", "Listing Languages"),
		locale.T("languages_cmd_description"),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, _ []string) error {
			return runner.Run()
		},
	).SetGroup(PlatformGroup).SetUnstable(true)
}

func newLanguageInstallCommand(prime *primer.Values) *captain.Command {
	runner := languages.NewUpdate(prime)

	params := languages.UpdateParams{}

	return captain.NewCommand(
		"install",
		locale.Tl("languages_install_title", "Installing Language"),
		locale.T("languages_install_cmd_description"),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        "language",
				Description: locale.T("arg_languages_install_description"),
				Required:    true,
				Value:       &params.Language,
			},
		},
		func(ccmd *captain.Command, _ []string) error {
			return runner.Run(&params)
		},
	)
}
