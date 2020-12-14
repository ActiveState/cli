package errs_test

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/rtutils"
)

func TestErrs(t *testing.T) {
	tests := []struct {
		name            string
		err             error
		wantMessage     string
		wantJoinMessage string
	}{
		{
			"Creates error",
			errs.New("hello %s", "world"),
			"hello world",
			"hello world",
		},
		{
			"Creates wrapped error",
			errs.Wrap(errors.New("Wrapped"), "Wrapper %s", "error"),
			"Wrapper error",
			"Wrapper error,Wrapped",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.err
			if err != nil && err.Error() != tt.wantMessage {
				t.Errorf("New() error message = %s, wantMessage %s", err.Error(), tt.wantMessage)
			}
			ee, ok := err.(errs.Errorable)
			if !ok {
				t.Fatalf("Error should be of type errs.Error")
			}
			if ee.Stack() == nil {
				t.Fatalf("Stacktrace was not created")
			}
			for i, frame := range ee.Stack().Frames {
				curFile := rtutils.CurrentFile()
				if strings.Contains(frame.Path, filepath.Dir(curFile)) && frame.Path != curFile {
					t.Fatalf("Stack should not contain reference to errs package.\nFound: %s at frame %d. Full stack:\n%s", frame.Path, i, ee.Stack().String())
				}
			}
			if joinmessage := errs.Join(tt.err, ","); joinmessage.Error() != tt.wantJoinMessage {
				t.Errorf("JoinMessage did not match, want: %s, got: %s", tt.wantJoinMessage, joinmessage.Error())
			}
		})
	}
}
