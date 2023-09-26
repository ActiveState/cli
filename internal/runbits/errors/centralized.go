package errors

import "github.com/ActiveState/cli/internal/errs"

type ErrNoProject struct {
	*errs.WrapperError
}

type ErrNotAuthenticated struct {
	*errs.WrapperError
}

type ErrActionAborted struct {
	*errs.WrapperError
}

type ErrPermission struct {
	*errs.WrapperError
	Details interface{}
}
