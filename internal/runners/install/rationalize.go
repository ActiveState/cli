package install

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/runbits/rationalizers"
	"github.com/ActiveState/cli/internal/sliceutils"
	bpResp "github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
	"github.com/ActiveState/cli/pkg/platform/model"
)

func (i *Install) rationalizeError(rerr *error) {
	var noMatchErr errNoMatches

	switch {
	case rerr == nil:
		return

	// No matches found
	case errors.As(*rerr, &noMatchErr):
		names := []string{}
		for _, r := range noMatchErr.requirements {
			names = append(names, fmt.Sprintf(`[ACTIONABLE]%s[/RESET]`, r.Requested.Name))
		}
		if len(noMatchErr.requirements) > 1 {
			*rerr = errs.WrapUserFacing(
				*rerr,
				locale.Tr("package_requirements_no_match", strings.Join(names, ", ")),
				errs.SetInput())
			return
		}
		suggestions, err := i.getSuggestions(noMatchErr.requirements[0], noMatchErr.languages)
		if err != nil {
			multilog.Error("Failed to retrieve suggestions: %v", err)
		}

		if len(suggestions) == 0 {
			*rerr = errs.WrapUserFacing(
				*rerr,
				locale.Tr("package_ingredient_alternatives_nosuggest", strings.Join(names, ", ")),
				errs.SetInput())
			return
		}

		*rerr = errs.WrapUserFacing(
			*rerr,
			locale.Tr("package_ingredient_alternatives", strings.Join(names, ", "), strings.Join(suggestions, "\n")),
			errs.SetInput())

	// Error staging a commit during install.
	case errors.As(*rerr, ptr.To(&bpResp.CommitError{})):
		rationalizers.HandleCommitErrors(rerr)

	}
}

func (i *Install) getSuggestions(req *requirement, languages []model.Language) ([]string, error) {
	var namespaces []string

	if req.Requested.Namespace == "" {
		for _, language := range languages {
			var ns string
			switch i.nsType {
			case model.NamespacePackage:
				ns = model.NewNamespacePackage(language.Name).String()
			case model.NamespaceBundle:
				ns = model.NewNamespaceBundle(language.Name).String()
			default:
				continue
			}
			namespaces = append(namespaces, ns)
		}
	} else {
		namespaces = []string{req.Requested.Namespace}
	}

	ingredients := make([]*model.IngredientAndVersion, 0)
	for _, namespace := range namespaces {
		results, err := model.SearchIngredients(namespace, req.Requested.Name, false, nil, i.prime.Auth())
		if err != nil {
			return nil, locale.WrapError(err, "err_package_ingredient_search", "Failed to resolve ingredient named: {{.V0}}", req.Requested.Name)
		}
		ingredients = append(ingredients, results...)
	}

	// Filter out irrelevant ingredients
	if req.Requested.Namespace == "" {
		// Filter out ingredients that don't target one of the supported languages
		ingredients = sliceutils.Filter(ingredients, func(iv *model.IngredientAndVersion) bool {
			if !model.NamespaceMatch(*iv.Ingredient.PrimaryNamespace, i.nsType.Matchable()) {
				return false
			}
			il := model.LanguageFromNamespace(*iv.Ingredient.PrimaryNamespace)
			for _, l := range languages {
				if l.Name == il {
					return true
				}
			}
			return false
		})
	}

	suggestions := []string{}
	for _, ing := range ingredients {
		suggestions = append(suggestions, fmt.Sprintf(" - %s/%s", *ing.Ingredient.PrimaryNamespace, *ing.Ingredient.Name))
	}

	maxResults := 5
	if len(suggestions) > maxResults {
		suggestions = suggestions[:maxResults]
	}

	return suggestions, nil
}
