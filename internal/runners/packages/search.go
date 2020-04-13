package packages

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// SearchRunParams tracks the info required for running search.
type SearchRunParams struct {
	Language  string
	ExactTerm bool
	Name      string
}

// Search manages the searching execution context.
type Search struct{}

// NewSearch prepares a searching execution context for use.
func NewSearch() *Search {
	return &Search{}
}

// Run is executed when `state packages search` is ran
func (s *Search) Run(params SearchRunParams) error {
	logging.Debug("ExecuteSearch")

	language, fail := targetedLanguage(params.Language)
	if fail != nil {
		return fail.WithDescription("package_err_cannot_obtain_language")
	}

	searchIngredients := model.SearchIngredients
	if params.ExactTerm {
		searchIngredients = model.SearchIngredientsStrict
	}

	packages, fail := searchIngredients(language, params.Name)
	if fail != nil {
		return fail.WithDescription("package_err_cannot_obtain_search_results")
	}
	if len(packages) == 0 {
		print.Line(locale.T("package_no_packages"))
		return nil
	}

	table := newPackagesTable(packages)
	sortByFirstTwoCols(table.data)

	print.Line(table.output())

	return nil
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
