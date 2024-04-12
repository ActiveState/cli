package revert

import (
	"errors"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
)

func rationalizeError(err *error) {
	if err == nil {
		return
	}

	var revertCommitError *response.RevertCommitError

	switch {
	case errors.As(*err, &revertCommitError):
		switch revertCommitError.Type {
		case types.NotFoundErrorType, types.ForbiddenErrorType:
			*err = errs.WrapUserFacing(*err,
				locale.Tl("err_revert_not_found",
					revertCommitError.Error(),
				),
				errs.SetInput(),
				errs.SetTips(
					locale.T("tip_private_project_auth"),
				),
			)
		case types.NoChangeSinceLastCommitErrorType:
			*err = errs.WrapUserFacing(*err,
				locale.Tl("err_revert_no_change",
					"Could not revert commit, no changes since last commit",
				),
				errs.SetInput(),
			)
		default:
			*err = errs.WrapUserFacing(*err,
				locale.Tl("err_revert_not_found",
					"Could not revert commit, recieved error message: {{.V0}}",
					revertCommitError.Error(),
				),
			)
		}
	}
}
