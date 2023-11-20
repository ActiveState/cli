package export

import (
	"errors"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
)

func rationalizeError(err *error) {
	switch {
	// export log with invalid --index.
	case errors.Is(*err, ErrInvalidLogIndex):
		*err = errs.WrapUserFacing(*err,
			locale.Tl("err_export_log_invalid_index", "Index must be >= 0"),
			errs.SetInput(),
		)

	// export log <prefix> with invalid <prefix>.
	case errors.Is(*err, ErrInvalidLogPrefix):
		*err = errs.WrapUserFacing(*err,
			locale.Tl("err_export_log_invalid_prefix", "Invalid log prefix"),
			errs.SetInput(),
			errs.SetTips(
				locale.Tl("export_log_prefix_tip", "Try a prefix like 'state' or 'state-svc'"),
			),
		)

	// export log does not turn up a log file.
	case errors.Is(*err, ErrLogNotFound):
		*err = errs.WrapUserFacing(*err,
			locale.Tl("err_export_log_out_of_bounds", "Log file not found"),
			errs.SetInput(),
		)
	}
}
