package builds

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

	switch {
	case err == nil:
		return

	case errors.Is(*err, rationalize.ErrNoProject):
		*err = errs.WrapUserFacing(*err,
			locale.Tr("err_no_project"),
			errs.SetInput())

	case errors.As(*err, &invalidCommitIdErr):
		*err = errs.WrapUserFacing(
			*err, locale.Tr("err_commit_id_invalid", invalidCommitIdErr.id),
			errs.SetInput())

	case errors.As(*err, &projectNotFoundErr):
		*err = errs.WrapUserFacing(*err,
			locale.Tr("err_api_project_not_found", projectNotFoundErr.Organization, projectNotFoundErr.Project),
			errs.SetIf(!auth.Authenticated(), errs.SetTips(locale.T("tip_private_project_auth"))),
			errs.SetInput())
	}

}
