package localx

import (
	"errors"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/osutils/stacktrace"
	"github.com/ActiveState/cli/internal/rtutils"
)

type L10n struct {
	key  string
	val  string
	args []string
}

func (l L10n) String() string {
	return locale.Tl(l.key, l.val, l.args...)
}

type UserErrorMsgs struct {
	Err  L10n
	Tips []L10n
}

type InputError struct {
	err error
}

func NewInputError(key, val string, args ...string) *InputError {
	return &InputError{NewError(key, val, args...)}
}

func WrapInputError(err error, key, val string, args ...string) *InputError {
	return &InputError{WrapError(err, key, val, args...)}
}

func (e *InputError) Error() string {
	return e.err.Error()
}

func (e *InputError) Unwrap() error {
	return e.err
}

type Error struct {
	err   error
	msgs  *UserErrorMsgs
	stack *stacktrace.Stacktrace
}

// Error is the error message
func (e *Error) Error() string {
	return e.err.Error()
}

func (e *Error) UserErrorMsgs() *UserErrorMsgs {
	return e.msgs
}

func (e *Error) Stack() *stacktrace.Stacktrace {
	return e.stack
}

// Unwrap returns the parent error, if applicable
func (e *Error) Unwrap() error {
	return e.err
}

func (e *Error) AddTip(key, val string, args ...string) {
	e.msgs.Tips = append(e.msgs.Tips, L10n{key, val, args})
}

func NewError(key, val string, args ...string) *Error {
	return WrapError(nil, key, val, args...)
}

func WrapError(err error, key, val string, args ...string) *Error {
	return &Error{
		err: err,
		msgs: &UserErrorMsgs{
			Err: L10n{key, val, args},
		},
		stack: stacktrace.GetWithSkip([]string{rtutils.CurrentFile()}),
	}
}

type UserErrorMessager interface {
	UserErrorMsgs() *UserErrorMsgs
}

func UserErrorMessages(err error) (msgs []*UserErrorMsgs) {
	for err != nil {
		if uem, ok := err.(UserErrorMessager); ok {
			msgs = append(msgs, uem.UserErrorMsgs())
		}
		err = errors.Unwrap(err)
	}
	return msgs
}
