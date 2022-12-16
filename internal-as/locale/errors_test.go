package locale_test

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal-as/errs"
	"github.com/ActiveState/cli/internal-as/locale"
	"github.com/ActiveState/cli/internal-as/rtutils"
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
