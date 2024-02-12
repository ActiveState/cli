package packages

import (
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/api/vulnerabilities/request"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	tea "github.com/charmbracelet/bubbletea"
)

// SearchRunParams tracks the info required for running search.
type SearchRunParams struct {
	Language   string
	ExactTerm  bool
	Ingredient captain.PackageValueNoVersion
	Timestamp  captain.TimeValue
}

// Search manages the searching execution context.
type Search struct {
	out  output.Outputer
	proj *project.Project
	auth *authentication.Auth
}

// NewSearch prepares a searching execution context for use.
func NewSearch(prime primeable) *Search {
	return &Search{
		out:  prime.Output(),
		proj: prime.Project(),
		auth: prime.Auth(),
	}
}

// Run is executed when `state packages search` is ran
func (s *Search) Run(params SearchRunParams, nstype model.NamespaceType) error {
	logging.Debug("ExecuteSearch")

	s.out.Notice(output.Title(locale.Tl("search_title", "Searching for: [ACTIONABLE]{{.V0}}[/RESET]", params.Ingredient.Name)))

	v, err := NewView()
	if err != nil {
		return errs.Wrap(err, "Could not create search view")
	}

	p := tea.NewProgram(v)

	var ns model.Namespace
	if params.Ingredient.Namespace == "" {
		language, err := targetedLanguage(params.Language, s.proj)
		if err != nil {
			return locale.WrapError(err, fmt.Sprintf("%s_err_cannot_obtain_language", nstype))
		}

		ns = model.NewNamespacePkgOrBundle(language, nstype)
	} else {
		ns = model.NewRawNamespace(params.Ingredient.Namespace)
	}

	var packages []*model.IngredientAndVersion
	if params.ExactTerm {
		packages, err = model.SearchIngredientsStrict(ns.String(), params.Ingredient.Name, true, true, params.Timestamp.Time)
	} else {
		packages, err = model.SearchIngredients(ns.String(), params.Ingredient.Name, true, params.Timestamp.Time)
	}
	if err != nil {
		return locale.WrapError(err, "package_err_cannot_obtain_search_results")
	}
	if len(packages) == 0 {
		return errs.AddTips(
			locale.NewInputError("err_search_no_"+ns.Type().String(), "", params.Ingredient.Name),
			locale.Tl("search_try_term", "Try a different search term"),
			locale.Tl("search_request_"+ns.Type().String(), ""),
		)
	}

	// The search endpoint will return all versions of a package, so we need to
	// use only the latest version of each package.
	seen := make(map[string]bool)
	var processedPackages []*model.IngredientAndVersion
	for _, pack := range packages {
		if pack.Ingredient.Name == nil {
			logging.Error("Package has no name: %v", pack)
			continue
		}

		if seen[*pack.Ingredient.Name] {
			continue
		}
		processedPackages = append(processedPackages, pack)
		seen[*pack.Ingredient.Name] = true
	}

	var vulns map[string][]*model.VulnerabilityIngredient
	if s.auth.Authenticated() {
		vulns, err = s.getVulns(processedPackages)
		if err != nil {
			return errs.Wrap(err, "Could not fetch vulnerabilities")
		}
	}

	table, err := createSearchTable(v.width, v.height, processedPackages, vulns)
	if err != nil {
		return errs.Wrap(err, "Could not create search table")
	}
	v.content = table.Content()
	v.packages = table.entries

	if s.out.Type().IsStructured() {
		s.out.Print(table)
		return nil
	}

	if _, err := p.Run(); err != nil {
		return errs.Wrap(err, "Failed to run search view")
	}

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

	commitID, err := localcommit.Get(proj.Dir())
	if err != nil {
		return "", errs.Wrap(err, "Unable to get local commit")
	}
	lang, err := model.LanguageByCommit(commitID)
	if err != nil {
		return "", errs.Wrap(err, "LanguageByCommit failed")
	}
	return lang.Name, nil
}

func (s *Search) getVulns(packages []*model.IngredientAndVersion) (map[string][]*model.VulnerabilityIngredient, error) {
	var ingredients []*request.Ingredient
	for _, pack := range packages {
		ingredients = append(ingredients, &request.Ingredient{
			Name:      *pack.Ingredient.Name,
			Namespace: *pack.Ingredient.PrimaryNamespace,
			Version:   pack.Version,
		})
	}

	vulns, err := model.FetchVulnerabilitiesForIngredients(s.auth, ingredients)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to fetch vulnerabilities")
	}

	vulnMap := make(map[string][]*model.VulnerabilityIngredient)
	for _, v := range vulns {
		key := ingredientVulnKey(v.PrimaryNamespace, v.Name, v.Version)
		vulnMap[key] = append(vulnMap[key], v)
	}

	return vulnMap, nil
}

func ingredientVulnKey(namespace, name, version string) string {
	return fmt.Sprintf("%s/%s/%s",
		strings.ToLower(namespace),
		strings.ToLower(name),
		strings.ToLower(version),
	)
}
