package pull

import (
	"errors"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
)

func rationalizeError(err *error) {
	if err == nil {
		return
	}

	var mergeCommitErr *model.MergedCommitError
	var noCommonParentErr *errNoCommonParent

	switch {
	case errors.As(*err, &mergeCommitErr):
		switch mergeCommitErr.Type {
		// Custom target does not have a compatible history
		case model.NoCommonBaseFoundType:
			*err = errs.WrapUserFacing(*err,
				locale.Tl("err_pull_no_common_base",
					"Could not merge, no common base found between local and remote commits",
				),
				errs.SetInput(),
			)
		case model.NotFoundErrorType, model.ForbiddenErrorType:
			*err = errs.WrapUserFacing(*err,
				locale.Tl("err_pull_not_found",
					mergeCommitErr.Error(),
				),
				errs.SetInput(),
				errs.SetTips(
					locale.T("tip_private_project_auth"),
				),
			)
		default:
			*err = errs.WrapUserFacing(*err,
				locale.Tl("err_pull_no_common_base",
					"Could not merge, recieved error message: {{.V0}}",
					mergeCommitErr.Error(),
				),
			)
		}
	case errors.As(*err, &noCommonParentErr):
		*err = errs.WrapUserFacing(*err,
			locale.Tr("err_pull_no_common_parent",
				noCommonParentErr.localCommitID.String(),
				noCommonParentErr.remoteCommitID.String(),
			),
			errs.SetInput(),
		)
	}
}
