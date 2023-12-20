package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
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
	).SetGroup(PlatformGroup).SetSupportsStructuredOutput().SetUnstable(true)
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
	).SetSupportsStructuredOutput()
}

func newLanguageSearchCommand(prime *primer.Values) *captain.Command {
	runner := languages.NewSearch(prime)

	return captain.NewCommand(
		"search",
		locale.Tl("languages_search_title", "Searching Languages"),
		locale.Tl("languages_search_cmd_description", "Search for an available language to use in your project"),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, _ []string) error {
			return runner.Run()
		},
	).SetSupportsStructuredOutput().SetUnstable(true)
}
