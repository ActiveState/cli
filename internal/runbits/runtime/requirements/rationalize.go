package requirements

import (
	"errors"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	runtime_runbit "github.com/ActiveState/cli/internal/runbits/runtime"
	bpResp "github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
	"github.com/ActiveState/cli/pkg/platform/model"
)

func (r *RequirementOperation) rationalizeError(err *error) {
	var tooManyMatchesErr *model.ErrTooManyMatches
	var noMatchesErr *ErrNoMatches
	var buildPlannerErr *bpResp.BuildPlannerError
	var resolveNamespaceErr *ResolveNamespaceError

	switch {
	case err == nil:
		return

	// Too many matches
	case errors.As(*err, &tooManyMatchesErr):
		*err = errs.WrapUserFacing(*err,
			locale.Tr("err_searchingredient_toomany", tooManyMatchesErr.Query),
			errs.SetInput())

	// No matches, and no alternate suggestions
	case errors.As(*err, &noMatchesErr) && noMatchesErr.Alternatives == nil:
		*err = errs.WrapUserFacing(*err,
			locale.Tr("package_ingredient_alternatives_nosuggest", noMatchesErr.Query),
			errs.SetInput())

	// No matches, but have alternate suggestions
	case errors.As(*err, &noMatchesErr) && noMatchesErr.Alternatives != nil:
		*err = errs.WrapUserFacing(*err,
			locale.Tr("package_ingredient_alternatives", noMatchesErr.Query, *noMatchesErr.Alternatives),
			errs.SetInput())

	// We communicate buildplanner errors verbatim as the intend is that these are curated by the buildplanner
	case errors.As(*err, &buildPlannerErr):
		*err = errs.WrapUserFacing(*err,
			buildPlannerErr.LocaleError(),
			errs.SetIf(buildPlannerErr.InputError(), errs.SetInput()))

	// Headless
	case errors.Is(*err, rationalize.ErrHeadless):
		*err = errs.WrapUserFacing(*err,
			locale.Tl(
				"err_requirement_headless",
				"Cannot update requirements for a headless project. Please visit {{.V0}} to convert your project and try again.",
				r.Project.URL(),
			),
			errs.SetInput())

	case errors.Is(*err, errNoRequirements):
		*err = errs.WrapUserFacing(*err,
			locale.Tl("err_no_requirements",
				"No requirements have been provided for this operation.",
			),
			errs.SetInput(),
		)

	case errors.As(*err, &resolveNamespaceErr):
		*err = errs.WrapUserFacing(*err,
			locale.Tl("err_resolve_namespace",
				"Could not resolve namespace for requirement '{{.V0}}'.",
				resolveNamespaceErr.Name,
			),
			errs.SetInput(),
		)

	case errors.Is(*err, errInitialNoRequirement):
		*err = errs.WrapUserFacing(*err,
			locale.T("err_initial_no_requirement"),
			errs.SetInput(),
		)

	case errors.Is(*err, errNoLanguage):
		*err = errs.WrapUserFacing(*err,
			locale.Tl("err_no_language", "Could not determine which language namespace to search for packages in. Please supply the language flag."),
			errs.SetInput(),
		)

	default:
		runtime_runbit.RationalizeSolveError(r.Project, r.Auth, err)

	}
}