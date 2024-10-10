package pull

import (
	"errors"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
)

func rationalizeError(err *error) {
	if err == nil {
		return
	}

	var mergeCommitErr *response.MergedCommitError
	var noCommonParentErr *errNoCommonParent
	var buildscriptMergeCommitErr *ErrBuildScriptMergeConflict

	switch {
	case errors.As(*err, &mergeCommitErr):
		switch mergeCommitErr.Type {
		// Custom target does not have a compatible history
		case types.NoCommonBaseFoundType:
			*err = errs.WrapUserFacing(*err,
				locale.Tl("err_pull_no_common_base",
					"Could not merge. No common base found between local and remote commits",
				),
				errs.SetInput(),
			)
		case types.NotFoundErrorType, types.ForbiddenErrorType:
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
					"Could not merge. Recieved error message: {{.V0}}",
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

	case errors.As(*err, &buildscriptMergeCommitErr):
		*err = errs.WrapUserFacing(*err,
			locale.Tl("err_build_script_merge",
				"Unable to automatically merge build scripts. Please resolve conflicts manually in '{{.V0}}' and then run '[ACTIONABLE]state commit[/RESET]'",
				filepath.Join(buildscriptMergeCommitErr.ProjectDir, constants.BuildScriptFileName)),
			errs.SetInput())
	}
}
