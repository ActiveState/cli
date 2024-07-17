package artifacts

import (
	"errors"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runbits/buildplanner"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	runtime_runbit "github.com/ActiveState/cli/internal/runbits/runtime"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

func rationalizeCommonError(proj *project.Project, auth *authentication.Auth, err *error) {
	var invalidCommitIdErr *errInvalidCommitId
	var projectNotFoundErr *model.ErrProjectNotFound
	var commitIdDoesNotExistInProject *buildplanner.ErrCommitDoesNotExistInProject

	switch {
	case errors.Is(*err, rationalize.ErrNoProject):
		*err = errs.WrapUserFacing(*err,
			locale.Tr("err_no_project"),
			errs.SetInput())

	case errors.As(*err, &invalidCommitIdErr):
		*err = errs.WrapUserFacing(
			*err, locale.Tr("err_commit_id_invalid_given", invalidCommitIdErr.Id),
			errs.SetInput())

	case errors.As(*err, &projectNotFoundErr):
		*err = errs.WrapUserFacing(*err,
			locale.Tr("err_api_project_not_found", projectNotFoundErr.Organization, projectNotFoundErr.Project),
			errs.SetIf(!auth.Authenticated(), errs.SetTips(locale.T("tip_private_project_auth"))),
			errs.SetInput())

	case errors.As(*err, &commitIdDoesNotExistInProject):
		*err = errs.WrapUserFacing(*err,
			locale.Tr("err_commit_id_not_in_history",
				commitIdDoesNotExistInProject.Project,
				commitIdDoesNotExistInProject.CommitID,
			),
			errs.SetInput())

	default:
		runtime_runbit.RationalizeSolveError(proj, auth, err)
		return

	}

}
