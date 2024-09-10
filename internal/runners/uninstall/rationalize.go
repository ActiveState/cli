package uninstall

import (
	"errors"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/runbits/rationalizers"
	bpResp "github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
)

func (u *Uninstall) rationalizeError(rerr *error) {
	var noMatchesErr *errNoMatches
	var multipleMatchesErr *errMultipleMatches

	switch {
	case rerr == nil:
		return

	case errors.As(*rerr, &noMatchesErr):
		*rerr = errs.WrapUserFacing(*rerr, locale.Tr("err_uninstall_nomatch", noMatchesErr.packages.String()))

	case errors.As(*rerr, &multipleMatchesErr):
		*rerr = errs.WrapUserFacing(*rerr, locale.Tr("err_uninstall_multimatch", multipleMatchesErr.packages.String()))

	// Error staging a commit during install.
	case errors.As(*rerr, ptr.To(&bpResp.CommitError{})):
		rationalizers.HandleCommitErrors(rerr)

	}
}
