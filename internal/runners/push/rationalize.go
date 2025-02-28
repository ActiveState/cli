package push

import (
	"errors"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	bpResp "github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
)

func rationalizeError(err *error) {
	if err == nil {
		return
	}

	var projectNameInUseErr *errProjectNameInUse

	var headlessErr *errHeadless

	var mergeCommitErr *bpResp.MergedCommitError

	switch {

	// Not authenticated
	case errors.Is(*err, rationalize.ErrNotAuthenticated):
		*err = errs.WrapUserFacing(*err,
			locale.T("err_push_not_authenticated"),
			errs.SetInput())

	// No activestate.yaml
	case errors.Is(*err, rationalize.ErrNoProject):
		*err = errs.WrapUserFacing(*err,
			locale.T("err_push_no_project"),
			errs.SetInput(),
			errs.SetTips(
				locale.T("push_push_tip_headless_init"),
				locale.T("push_push_tip_headless_cwd"),
			))

	case errors.As(*err, &headlessErr):
		*err = errs.WrapUserFacing(*err,
			locale.Tr("err_push_headless", headlessErr.ProjectURL),
			errs.SetInput(),
		)

	// No commits made yet
	case errors.Is(*err, errNoCommit):
		*err = errs.WrapUserFacing(*err,
			locale.T("err_push_nocommit"),
			errs.SetInput(),
		)

	// Project name is already in use
	case errors.As(*err, &projectNameInUseErr):
		*err = errs.WrapUserFacing(*err,
			locale.Tr("err_push_create_nonunique", projectNameInUseErr.Namespace.String()),
			errs.SetInput(),
		)

	// Project creation aborted
	case errors.Is(*err, rationalize.ErrActionAborted):
		*err = errs.WrapUserFacing(*err,
			locale.T("err_push_create_project_aborted"),
			errs.SetInput())

	case errors.As(*err, &mergeCommitErr):
		switch mergeCommitErr.Type {
		// Need to pull first
		case types.FastForwardErrorType:
			*err = errs.WrapUserFacing(*err,
				locale.T("err_push_outdated"),
				errs.SetInput(),
				errs.SetTips(locale.T("err_tip_push_outdated")))

			// Custom target does not have a compatible history
		case types.NoCommonBaseFoundType:
			*err = errs.WrapUserFacing(*err,
				locale.T("err_push_target_invalid_history"),
				errs.SetInput())
		}
	}
}
