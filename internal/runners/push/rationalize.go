package push

import (
	"errors"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
)

func rationalizeError(err *error) {
	if err == nil {
		return
	}

	var projectNameInUseErr *errProjectNameInUse

	switch {

	// Not authenticated
	case errors.Is(*err, rationalize.ErrNotAuthenticated):
		*err = errs.WrapUserFacing(*err,
			locale.T("err_push_not_authenticated"),
			errs.SetInput())

	// No activestate.yaml
	case errors.Is(*err, rationalize.ErrNoProject):
		*err = errs.WrapUserFacing(*err,
			locale.T("err_push_headless"),
			errs.SetInput(),
			errs.SetTips(
				locale.T("push_push_tip_headless_init"),
				locale.T("push_push_tip_headless_cwd"),
			))

	// No commits made yet
	case errors.Is(*err, errNoCommit):
		*err = errs.WrapUserFacing(*err,
			locale.T("err_push_nocommit"),
			errs.SetInput(),
		)

	// No changes made
	case errors.Is(*err, errNoChanges):
		*err = errs.WrapUserFacing(*err,
			locale.T("push_no_changes"),
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

	// Custom target does not have a compatible history
	case errors.Is(*err, errTargetInvalidHistory):
		*err = errs.WrapUserFacing(*err,
			locale.T("err_push_target_invalid_history"),
			errs.SetInput())

	// Need to pull first
	case errors.Is(*err, errPullNeeded):
		*err = errs.WrapUserFacing(*err,
			locale.T("err_push_outdated"),
			errs.SetInput(),
			errs.SetTips(locale.T("err_tip_push_outdated")))
	}
}
