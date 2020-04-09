package packages

import (
	"github.com/spf13/cobra"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// SearchArgs holds the arg values passed through the command line
var SearchArgs struct {
	Name string
}

// SearchFlags holds the search-related flag values passed through the command line
var SearchFlags struct {
	Language  string
	ExactTerm bool
}

// SearchCommand is the `packages search` command struct
var SearchCommand = &commands.Command{
	Name:        "search",
	Description: "package_search_description",

	Arguments: []*commands.Argument{
		{
			Name:        "package_arg_name",
			Description: "package_arg_name_description",
			Variable:    &SearchArgs.Name,
			Required:    true,
		},
	},
	Flags: []*commands.Flag{
		{
			Name:        "language",
			Description: "package_search_flag_language_description",
			Type:        commands.TypeString,
			StringVar:   &SearchFlags.Language,
		},
		{
			Name:        "exact-term",
			Description: "package_search_flag_exact-term_description",
			Type:        commands.TypeBool,
			BoolVar:     &SearchFlags.ExactTerm,
		},
	},
}

func init() {
	SearchCommand.Run = ExecuteSearch // Work around initialization loop
}

// ExecuteSearch is executed when `state packages search` is ran
func ExecuteSearch(cmd *cobra.Command, allArgs []string) {
	logging.Debug("ExecuteSearch")

	language, fail := targetedLanguage(SearchFlags.Language)
	if fail != nil {
		failures.Handle(fail, locale.T("package_err_cannot_obtain_language"))
		return
	}

	searchIngredients := model.SearchIngredients
	if SearchFlags.ExactTerm {
		searchIngredients = model.SearchIngredientsStrict
	}

	packages, fail := searchIngredients(language, SearchArgs.Name)
	if fail != nil {
		failures.Handle(fail, locale.T("package_err_cannot_obtain_search_results"))
		return
	}
	if len(packages) == 0 {
		print.Line(locale.T("package_no_packages"))
		return
	}

	table := newPackagesTable(packages)
	sortByFirstTwoCols(table.data)

	print.Line(table.output())
}

func targetedLanguage(languageOpt string) (string, *failures.Failure) {
	if languageOpt != "" {
		return languageOpt, nil
	}

	proj, fail := project.GetSafe()
	if fail != nil {
		return "", fail
	}

	return model.DefaultLanguageForProject(proj.Owner(), proj.Name())
}

func newPackagesTable(packages []*model.IngredientAndVersion) *table {
	if packages == nil {
		return nil
	}

	headers := []string{
		locale.T("package_name"),
		locale.T("package_version"),
	}

	filterNilStr := func(s *string) string {
		if s == nil {
			return ""
		}
		return *s
	}

	rows := make([][]string, 0, len(packages))
	for _, pack := range packages {
		row := []string{
			filterNilStr(pack.Ingredient.Name),
			filterNilStr(pack.Version.Version),
		}
		rows = append(rows, row)
	}

	return newTable(headers, rows)
}
