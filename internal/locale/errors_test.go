package locale_test

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/rtutils"
)

func TestIsError(t *testing.T) {
	tests := []struct {
		name            string
		err             error
		wantMessage     string
		wantJoinMessage string
		isError         bool
		isInputError    bool
	}{
		{
			"NewError from ID",
			locale.NewError("id_error"),
			"id_error",
			"id_error",
			true,
			false,
		},
		{
			"NewError",
			locale.NewError("", "Localized {{.V0}}", "Error"),
			"Localized Error",
			"Localized Error",
			true,
			false,
		},
		{
			"WrapError",
			locale.WrapError(errors.New("Error Value"), "", "Localized {{.V0}}", "Error"),
			"Localized Error",
			"Localized Error",
			true,
			false,
		},
		{
			"NewInputError",
			locale.NewInputError("", "Localized Error"),
			"Localized Error",
			"Localized Error",
			true,
			true,
		},
		{
			"WrapError over NewInputError",
			locale.WrapError(locale.NewInputError("", "Input error"), "", "Wrapper"),
			"Wrapper",
			"Wrapper,Input error",
			true,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.wantMessage {
				t.Errorf("err.Error() = %v, want %v", tt.err.Error(), tt.wantMessage)
			}
			if got := locale.IsError(tt.err); got != tt.isError {
				t.Errorf("IsError() = %v, want %v", got, tt.isError)
			}
			if got := locale.IsInputError(tt.err); got != tt.isInputError {
				t.Errorf("IsInputError() = %v, want %v", got, tt.isInputError)
			}

			if tt.isError {
				if joinmessage := locale.JoinErrors(tt.err, ","); joinmessage.Error() != tt.wantJoinMessage {
					t.Errorf("JoinMessage did not match, want: %s, got: %s", tt.wantJoinMessage, joinmessage.Error())
				}
				ee, ok := tt.err.(errs.Errorable)
				if !ok {
					t.Error("Error should be of type errs.Error")
					t.FailNow()
				}
				if ee.Stack() == nil {
					t.Error("Stacktrace was not created")
					t.FailNow()
				}
				for i, frame := range ee.Stack().Frames {
					curFile := rtutils.CurrentFile()
					if strings.Contains(frame.Path, filepath.Dir(curFile)) && frame.Path != curFile {
						t.Errorf("Stack should not contain reference to locale package.\nFound: %s at frame %d. Full stack:\n%s", frame.Path, i, ee.Stack().String())
						t.FailNow()
					}
				}
			}
		})
	}
}

func TestUnwrapError(t *testing.T) {
	errPlain := errors.New("plain error")
	errLocalized := locale.NewError("localized error")
	errLocalized2 := locale.NewError("localized error 2")
	errLocalizedForWrapWithLocale := locale.NewError("localized error for wrap with locale")
	errLocaleWrapWithPlain := locale.WrapError(errPlain, "wrapped localized error")
	errPlainWrapWithLocale := errs.Wrap(errLocalizedForWrapWithLocale, "wrapped plain error")
	errMultiWithLocaleWrap := errs.Combine(errPlain, errPlainWrapWithLocale)
	errMulti := errs.Combine(errLocalized, errLocalized2, errPlain, errPlainWrapWithLocale, errLocaleWrapWithPlain)
	errPlainWrappedMulti := errs.Wrap(errMulti, "wrapped plain error")

	tests := []struct {
		name       string
		inError    error
		wantErrors []error
	}{
		{
			"Plain",
			errPlain,
			[]error{},
		},
		{
			"Localized",
			errLocalized,
			[]error{errLocalized},
		},
		{
			"Localized wrapped with plain",
			errLocaleWrapWithPlain,
			[]error{errLocalized},
		},
		{
			"Plain wrapped with localized",
			errPlainWrapWithLocale,
			[]error{errLocalizedForWrapWithLocale},
		},
		{
			"Multi error",
			errMulti,
			[]error{errLocalized, errLocalized2},
		},
		{
			"Plain wrapped Multi error",
			errPlainWrappedMulti,
			[]error{errLocalized, errLocalized2, errLocalizedForWrapWithLocale, errLocaleWrapWithPlain},
		},
		{
			"Multi error with locale wrap",
			errMultiWithLocaleWrap,
			[]error{errLocalizedForWrapWithLocale},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := locale.UnwrapError(tt.inError)

			if len(got) != len(tt.wantErrors) {
				t.Errorf("UnwrapError() has %d results: %v, want %d results: %v", len(got), got, len(tt.wantErrors), tt.wantErrors)
			}

			for n, wantErr := range tt.wantErrors {
				if got[n].Error() != wantErr.Error() {
					t.Errorf("Resulting error: %s, did not match: %s", got[n].Error(), wantErr.Error())
				}
			}
		})
	}
}
