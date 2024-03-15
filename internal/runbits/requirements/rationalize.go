package requirements

import (
	"errors"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	bpModel "github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	"github.com/ActiveState/cli/pkg/platform/model"
)

func (r *RequirementOperation) rationalizeError(err *error) {
	var tooManyMatchesErr *model.ErrTooManyMatches
	var noMatchesErr *ErrNoMatches
	var buildPlannerErr *bpModel.BuildPlannerError

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
			buildPlannerErr.LocalizedError(),
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
	}
}
