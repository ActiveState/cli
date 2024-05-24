package runtime

import "github.com/ActiveState/cli/internal/errs"

var (
	ErrNoPlatformMatch = errs.New("Current platform does not match any of the runtime platforms")
)

// ProgressReportError designates an error in the event handler for reporting progress.
type ProgressReportError struct {
	*errs.WrapperError
}
