package errs_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/rtutils"
	"github.com/hashicorp/go-multierror"
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
		{
			"combined errors 1",
			args{
				errs.Combine(&exec.ExitError{&os.ProcessState{}, []byte("")}, errs.New("Random")),
				&exec.ExitError{},
			},
			true,
		},
		{
			"combined errors 2 - inverted",
			args{
				errs.Combine(errs.New("Random"), &exec.ExitError{&os.ProcessState{}, []byte("")}),
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

func TestAddTips(t *testing.T) {
	type args struct {
		err  error
		tips []string
	}
	tests := []struct {
		name          string
		args          args
		wantErrorMsgs []string
		wantTips      []string
	}{
		{
			"Simple",
			args{
				errs.New("error"),
				[]string{"tip1", "tip2"},
			},
			[]string{"error"},
			[]string{"tip1", "tip2"},
		},
		{
			"Localized",
			args{
				locale.NewError("error"),
				[]string{"tip1", "tip2"},
			},
			[]string{"error"},
			[]string{"tip1", "tip2"},
		},
		{
			"Multi error",
			args{
				errs.Combine(errs.New("error1"), errs.New("error2")),
				[]string{"tip1", "tip2"},
			},
			[]string{"error1", "error2"},
			[]string{"tip1", "tip2"},
		},
		{
			"Multi error with locale",
			args{
				errs.Combine(locale.NewError("error1"), locale.NewError("error2")),
				[]string{"tip1", "tip2"},
			},
			[]string{"error1", "error2"},
			[]string{"tip1", "tip2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errs.AddTips(tt.args.err, tt.args.tips...)
			gotTips := []string{}
			msgs := []string{}
			for err != nil {
				_, isMultiError := err.(*multierror.Error)
				if !isMultiError && err.Error() != errs.TipMessage {
					msgs = append(msgs, err.Error())
				}

				// Check via direct type cast or via direct `.As()` method because otherwise the unwrapper of
				// errors.As will go down paths we're not interested in.
				var errTips errs.ErrorTips
				if x, ok := err.(interface{ As(interface{}) bool }); ok && x.As(&errTips) {
					gotTips = append(gotTips, errTips.ErrorTips()...)
				} else if x, ok := err.(errs.ErrorTips); ok {
					gotTips = append(gotTips, x.ErrorTips()...)
				}

				err = errors.Unwrap(err)
			}
			if !reflect.DeepEqual(gotTips, tt.wantTips) {
				t.Errorf("AddTips() = %v, want %v", gotTips, tt.wantTips)
			}
			if !reflect.DeepEqual(msgs, tt.wantErrorMsgs) {
				t.Errorf("Error Msgs = %v, want %v", msgs, tt.wantErrorMsgs)
			}
		})
	}
}
