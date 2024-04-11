package packages

import (
	"fmt"

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

	ts, err := getTime(&params.Timestamp, s.auth, s.proj)
	if err != nil {
		return errs.Wrap(err, "Unable to get timestamp from params")
	}

	var packages []*model.IngredientAndVersion
	if params.ExactTerm {
		packages, err = model.SearchIngredientsLatestStrict(ns.String(), params.Ingredient.Name, true, true, ts)
	} else {
		packages, err = model.SearchIngredientsLatest(ns.String(), params.Ingredient.Name, true, ts)
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

	var vulns []*model.VulnerabilityIngredient
	if s.auth.Authenticated() {
		vulns, err = s.getVulns(packages)
		if err != nil {
			return errs.Wrap(err, "Could not fetch vulnerabilities")
		}
	}

	results, err := createSearchResults(packages, vulns)
	if err != nil {
		return errs.Wrap(err, "Could not create search table")
	}

	if s.out.Type().IsStructured() || !s.out.Config().Interactive {
		s.out.Print(results)
		return nil
	}

	v, err := NewView(results, s.out)
	if err != nil {
		return errs.Wrap(err, "Could not create search view")
	}

	p := tea.NewProgram(v)

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

func (s *Search) getVulns(packages []*model.IngredientAndVersion) ([]*model.VulnerabilityIngredient, error) {
	var ingredients []*request.Ingredient
	for _, pkg := range packages {
		ingredients = append(ingredients, &request.Ingredient{
			Name:      *pkg.Ingredient.Name,
			Namespace: *pkg.Ingredient.PrimaryNamespace,
			Version:   pkg.Version,
		})
	}

	return model.FetchVulnerabilitiesForIngredients(s.auth, ingredients)
}
