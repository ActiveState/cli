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
	bpResp "github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
)

func rationalizeError(auth *authentication.Auth, rerr *error) {
	var noMatchErr errNoMatches

	switch {
	case rerr == nil:
		return

	// No matches found
	case errors.As(*rerr, &noMatchErr):
		names := []string{}
		for _, r := range noMatchErr.requirements {
			names = append(names, fmt.Sprintf(`"[[ACTIONABLE]%s[/RESET]"`, r.input.Name))
		}
		if len(noMatchErr.requirements) > 1 {
			*rerr = errs.WrapUserFacing(*rerr, locale.Tr("package_requirements_no_match", strings.Join(names, ", ")))
			return
		}
		suggestions, err := getSuggestions(noMatchErr.requirements[0], auth)
		if err != nil {
			multilog.Error("Failed to retrieve suggestions: %v", err)
		}

		if len(suggestions) == 0 {
			*rerr = errs.WrapUserFacing(*rerr, locale.Tr("package_ingredient_alternatives_nosuggest", strings.Join(names, ", ")))
			return
		}

		*rerr = errs.WrapUserFacing(*rerr, locale.Tr("package_ingredient_alternatives", strings.Join(names, ", ")))

	// Error staging a commit during install.
	case errors.As(*rerr, ptr.To(&bpResp.CommitError{})):
		rationalizers.HandleCommitErrors(rerr)

	}
}

func getSuggestions(req *requirement, auth *authentication.Auth) ([]string, error) {
	results, err := model.SearchIngredients(req.input.Namespace, req.input.Name, false, nil, auth)
	if err != nil {
		return []string{}, locale.WrapError(err, "package_ingredient_err_search", "Failed to resolve ingredient named: {{.V0}}", req.input.Name)
	}

	maxResults := 5
	if len(results) > maxResults {
		results = results[:maxResults]
	}

	suggestions := make([]string, 0, maxResults+1)
	for _, result := range results {
		suggestions = append(suggestions, fmt.Sprintf(" - %s", *result.Ingredient.Name))
	}

	return suggestions, nil
}
