package packages

import (
	"fmt"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
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
type Search struct {
	out output.Outputer
}

type searchPackageRow struct {
	Pkg           string `json:"package" locale:"package_name,Name"`
	Version       string `json:"version" locale:"package_version,Latest Version"`
	OlderVersions string `json:"versions" locale:","`
	versions      int
}

// NewSearch prepares a searching execution context for use.
func NewSearch(prime primer.Outputer) *Search {
	return &Search{
		out: prime.Output(),
	}
}

// Run is executed when `state packages search` is ran
func (s *Search) Run(params SearchRunParams, pt PackageType) error {
	logging.Debug("ExecuteSearch")

	language, fail := targetedLanguage(params.Language)
	if fail != nil {
		return fail.WithDescription(fmt.Sprintf("%s_err_cannot_obtain_language", pt.String()))
	}

	searchIngredients := model.SearchIngredients
	if params.ExactTerm {
		searchIngredients = model.SearchIngredientsStrict
	}

	packages, fail := searchIngredients(pt.Namespace(), language, params.Name)
	if fail != nil {
		return fail.WithDescription("package_err_cannot_obtain_search_results")
	}
	if len(packages) == 0 {
		return errs.AddTips(
			locale.NewInputError("err_package_search_no_packages", `No packages in our catalogue match [NOTICE]"{{.V0}}"[/RESET].`, params.Name),
			locale.Tl("search_try_term", "Try a different search term"),
			locale.Tl("search_request", "Request a package at [ACTIONABLE]https://community.activestate.com/[/RESET]"),
		)

	}
	results := formatSearchResults(packages, pt)
	s.out.Print(results)

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

	return model.LanguageForCommit(proj.CommitUUID())
}

func formatSearchResults(packages []*model.IngredientAndVersion, pt PackageType) []searchPackageRow {
	var rows []searchPackageRow

	filterNilStr := func(s *string) string {
		if s == nil {
			return ""
		}
		return *s
	}

	for _, pack := range packages {
		row := searchPackageRow{
			Pkg:      filterNilStr(pack.Ingredient.Name),
			Version:  pack.Version,
			versions: len(pack.Versions),
		}
		rows = append(rows, row)
	}

	return mergeSearchRows(rows)
}

func mergeSearchRows(rows []searchPackageRow) []searchPackageRow {
	var mergedRows []searchPackageRow
	var name string
	for _, row := range rows {
		// The search API returns results sorted by name and then descending version
		// so we can use the first unique value as our latest version
		if name == row.Pkg {
			continue
		}
		name = row.Pkg

		newRow := searchPackageRow{
			Pkg:      row.Pkg,
			Version:  row.Version,
			versions: row.versions,
		}

		if row.versions > 1 {
			newRow.OlderVersions = fmt.Sprintf("+ %d older versions", row.versions-1)
		}
		mergedRows = append(mergedRows, newRow)
	}

	return mergedRows
}
