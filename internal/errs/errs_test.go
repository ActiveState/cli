package errs_test

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/rtutils"
)

func TestNew(t *testing.T) {
	type args struct {
		message string
		args    []interface{}
	}
	tests := []struct {
		name        string
		args        args
		wantMessage string
	}{
		{
			"Creates error, no args",
			args{
				"Error Message",
				[]interface{}{},
			},
			"Error Message",
		},
		{
			"Creates error, with args",
			args{
				"Error Message %d %s",
				[]interface{}{1, "string"},
			},
			"Error Message 1 string",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errs.New(tt.args.message, tt.args.args...)
			if err != nil && err.Error() != tt.wantMessage {
				t.Errorf("New() error message = %s, wantMessage %s", err.Error(), tt.wantMessage)
			}
			ee, ok := err.(*errs.Error)
			if ! ok {
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
					t.Errorf("Stack should not contain reference to errs package.\nFound: %s at frame %d. Full stack:\n%s", frame.Path, i, ee.Stack().String())
					t.FailNow()
				}
			}
			if ee.Unwrap() != nil {
				t.Errorf("Newly created error should not unwrap to anything, unwrapped to: %v", ee.Unwrap())
			}
		})
	}
}

func TestNewWrapped(t *testing.T) {
	type args struct {
		err        error
		wrapTarget error
	}
	tests := []struct {
		name        string
		args        args
		wantMessage string
	}{
		{
			"Creates wrapped error, no args",
			args{
				fmt.Errorf("Error Message"),
				errs.UserInputErr,
			},
			"Error Message",
		},
		{
			"Creates error, with args",
			args{
				fmt.Errorf("Error Message %d %s", 1, "string"),
				errs.UserInputErr,
			},
			"Error Message 1 string",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errs.Wrap(tt.args.err, tt.args.wrapTarget)
			if err == nil {
				return
			}
			if err.Error() != tt.wantMessage {
				t.Errorf("NewWrapped() error message = %s, wantMessage %s", err.Error(), tt.wantMessage)
			}
			ee, ok := err.(*errs.Error)
			if ! ok {
				t.Error("Error should be of type errs.Error")
			}
			if ee.Unwrap() != tt.args.wrapTarget {
				t.Errorf("error did not unwrap to provided error")
			}
			if ! errors.Is(err, tt.args.wrapTarget) {
				t.Errorf("error should wrap provided error")
			}
		})
	}
}
