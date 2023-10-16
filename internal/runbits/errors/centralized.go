package errors

import "github.com/ActiveState/cli/internal/errs"

type ErrNoProject struct {
	*errs.WrapperError
}
