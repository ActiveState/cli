package packages

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
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
	out  output.Outputer
	proj *project.Project
}

// NewSearch prepares a searching execution context for use.
func NewSearch(prime primeable) *Search {
	return &Search{
		out:  prime.Output(),
		proj: prime.Project(),
	}
}

// Run is executed when `state packages search` is ran
func (s *Search) Run(params SearchRunParams, nstype model.NamespaceType) error {
	logging.Debug("ExecuteSearch")

	language, err := targetedLanguage(params.Language, s.proj)
	if err != nil {
		return locale.WrapError(err, fmt.Sprintf("%s_err_cannot_obtain_language", nstype))
	}

	ns := model.NewNamespacePkgOrBundle(language, nstype)

	searchIngredients := model.SearchIngredients
	if params.ExactTerm {
		searchIngredients = model.SearchIngredientsStrict
	}

	packages, err := searchIngredients(ns, params.Name)
	if err != nil {
		return locale.WrapError(err, "package_err_cannot_obtain_search_results")
	}
	if len(packages) == 0 {
		return errs.AddTips(
			locale.NewInputError("err_package_search_no_packages", `No packages in our catalogue match [NOTICE]"{{.V0}}"[/RESET].`, params.Name),
			locale.Tl("search_try_term", "Try a different search term"),
			locale.Tl("search_request", "Request a package at [ACTIONABLE]https://community.activestate.com/[/RESET]"),
		)
	}
	results := formatSearchResults(packages)
	s.out.Print(results)

	return nil
}

func targetedLanguage(languageOpt string, proj *project.Project) (string, error) {
	if languageOpt != "" {
		return languageOpt, nil
	}
	if proj == nil {
		return "", locale.NewInputError(
			"err_no_language_derived",
			"Language must be provided by flag or by running this command within a project.",
		)
	}

	return model.LanguageForCommit(proj.CommitUUID())
}

type modules []string

func makeModules(normalizedName string, pack *model.IngredientAndVersion) modules {
	var ms modules
	for _, module := range pack.LatestVersion.ProvidedFeatures {
		if module.Feature != nil && *module.Feature != normalizedName {
			ms = append(ms, *module.Feature)
		}

	}
	return ms
}

func (ms modules) String() string {
	if len(ms) == 0 {
		return ""
	}

	var b strings.Builder

	b.WriteString("[DISABLED]")
	b.WriteString(locale.Tl("title_matching_modules", "Matching modules"))
	b.WriteRune('\n')

	prefix := '├'
	for i, module := range ms {
		if i == len(ms)-1 {
			prefix = '└'
		}

		b.WriteRune(prefix)
		b.WriteString("─ ")
		b.WriteString(module)
		b.WriteRune('\n')
	}

	b.WriteRune('\n')
	b.WriteString("[/RESET]")

	return b.String()
}

type searchPackageRow struct {
	Pkg           string `json:"package" locale:"package_name,Name"`
	Version       string `json:"version" locale:"package_version,Latest Version"`
	OlderVersions string `json:"versions" locale:","`
	versions      int
	Modules       modules `json:"matching_modules,omitempty" opts:"emptyNil,separateLine,shiftCols=1"`
}

func formatSearchResults(packages []*model.IngredientAndVersion) []searchPackageRow {
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
			Modules:  makeModules(pack.Ingredient.NormalizedName, pack),
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
			Modules:  row.Modules,
		}

		if row.versions > 1 {
			olderVersions := row.versions - 1
			newRow.OlderVersions = locale.Tl("search_older_versions", "+ {{.V0}} older versions", strconv.Itoa(olderVersions))
		}
		mergedRows = append(mergedRows, newRow)
	}

	return mergedRows
}
