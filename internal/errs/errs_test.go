package errs_test

import (
	"errors"
	"os"
	"os/exec"
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

type standardError struct{ error }

func TestMatches(t *testing.T) {
	type args struct {
		err    error
		target interface{}
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"Simple match",
			args{
				&standardError{errors.New("error")},
				&standardError{},
			},
			true,
		},
		{
			"Simple miss-match",
			args{
				errors.New("error"),
				&standardError{},
			},
			false,
		},
		{
			"Wrapped match",
			args{
				errs.Wrap(&standardError{errors.New("error")}, "Wrapped"),
				&standardError{},
			},
			true,
		},
		{
			"exec.ExitError", // this one has proved troublesome
			args{
				&exec.ExitError{&os.ProcessState{}, []byte("")},
				&exec.ExitError{},
			},
			true,
		},
		{
			"wrapped exec.ExitError",
			args{
				errs.Wrap(&exec.ExitError{&os.ProcessState{}, []byte("")}, "wrapped"),
				&exec.ExitError{},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errs.Matches(tt.args.err, tt.args.target); got != tt.want {
				t.Errorf("Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}