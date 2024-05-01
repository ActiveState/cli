package locale

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/osutils/stacktrace"
	"github.com/ActiveState/cli/internal/rtutils"
)

type ExternalError interface {
	ExternalError() bool
}

func IsExternalError(err error) bool {
	if err == nil {
		return false
	}

	for _, err := range errs.Unpack(err) {
		errExternal, ok := err.(ExternalError)
		if ok && errExternal.ExternalError() {
			return true
		}
	}

	return false
}

func NewExternalError(id string, args ...string) *LocalizedError {
	return WrapExternalError(nil, id, args...)
}

func WrapExternalError(wrapTarget error, id string, args ...string) *LocalizedError {
	locale := id
	if len(args) > 0 {
		locale, args = args[0], args[1:]
	}
	if locale == "" {
		locale = id
	}

	l := &LocalizedError{}
	translation := Tl(id, locale, args...)
	l.externalErr = true
	l.wrapped = wrapTarget
	l.localized = translation
	l.stack = stacktrace.GetWithSkip([]string{rtutils.CurrentFile()})

	return l
}
