package rationalizers

import (
	"errors"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	bpResp "github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
)

func HandleCommitErrors(rerr *error) {
	var commitError *bpResp.CommitError
	if !errors.As(*rerr, &commitError) {
		return
	}
	switch commitError.Type {
	case types.NotFoundErrorType:
		*rerr = errs.WrapUserFacing(*rerr,
			locale.Tl("err_packages_not_found", "Could not make runtime changes because your project was not found."),
			errs.SetInput(),
			errs.SetTips(locale.T("tip_private_project_auth")),
		)
	case types.ForbiddenErrorType:
		*rerr = errs.WrapUserFacing(*rerr,
			locale.Tl("err_packages_forbidden", "Could not make runtime changes because you do not have permission to do so."),
			errs.SetInput(),
			errs.SetTips(locale.T("tip_private_project_auth")),
		)
	case types.HeadOnBranchMovedErrorType:
		*rerr = errs.WrapUserFacing(*rerr,
			locale.T("err_buildplanner_head_on_branch_moved"),
			errs.SetInput(),
		)
	case types.NoChangeSinceLastCommitErrorType:
		*rerr = errs.WrapUserFacing(*rerr,
			locale.Tl("err_packages_exist", "The requested package is already installed."),
			errs.SetInput(),
		)
	default:
		*rerr = errs.WrapUserFacing(*rerr,
			locale.Tl("err_packages_buildplanner_error", "Could not make runtime changes due to the following error: {{.V0}}", commitError.Message),
			errs.SetInput(),
		)
	}
}
