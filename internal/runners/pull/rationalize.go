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
		}
	}
}
