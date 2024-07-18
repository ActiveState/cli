package artifacts

import (
	"errors"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
)

func rationalizeCommonError(err *error, auth *authentication.Auth) {
	var invalidCommitIdErr *errInvalidCommitId
	var projectNotFoundErr *model.ErrProjectNotFound
	var commitIdDoesNotExistInProject *errCommitDoesNotExistInProject

	switch {
	case errors.Is(*err, rationalize.ErrNoProject):
		*err = errs.WrapUserFacing(*err,
			locale.Tr("err_no_project"),
			errs.SetInput())

	case errors.As(*err, &invalidCommitIdErr):
		*err = errs.WrapUserFacing(
			*err, locale.Tr("err_commit_id_invalid_given", invalidCommitIdErr.id),
			errs.SetInput())

	case errors.As(*err, &projectNotFoundErr):
		*err = errs.WrapUserFacing(*err,
			locale.Tr("err_api_project_not_found", projectNotFoundErr.Organization, projectNotFoundErr.Project),
			errs.SetIf(!auth.Authenticated(), errs.SetTips(locale.T("tip_private_project_auth"))),
			errs.SetInput())

	case errors.As(*err, &commitIdDoesNotExistInProject):
		*err = errs.WrapUserFacing(*err,
			locale.Tl("err_commit_id_not_in_history",
				"The project '[ACTIONABLE]{{.V0}}[/RESET]' does not contain the provided commit: '[ACTIONABLE]{{.V1}}[/RESET]'.",
				commitIdDoesNotExistInProject.Project,
				commitIdDoesNotExistInProject.CommitID,
			),
			errs.SetInput())
	}

}
