package errs_test

import (
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/rtutils"
	"github.com/stretchr/testify/assert"
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
			"Wrapper error: Wrapped",
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
			if joinmessage := errs.JoinMessage(err); joinmessage != tt.wantJoinMessage {
				t.Errorf("JoinMessage did not match, want: '%s', got: '%s'", tt.wantJoinMessage, joinmessage)
			}
		})
	}
}

type standardError struct{ error }

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
				errs.Pack(errs.New("error1"), errs.New("error2")),
				[]string{"tip1", "tip2"},
			},
			[]string{"error1", "error2"},
			[]string{"tip1", "tip2"},
		},
		{
			"Multi error with locale",
			args{
				errs.Pack(locale.NewError("error1"), locale.NewError("error2")),
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
			errors := errs.Unpack(err)
			for _, err := range errors {
				_, isMultiError := err.(*errs.PackedErrors)
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

func TestUnpack(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want []string
	}{
		{
			"Single",
			errs.New("error1"),
			[]string{"error1"},
		},
		{
			"Wrapped",
			errs.Wrap(errs.New("error1"), "error2"),
			[]string{"error2", "error1"},
		},
		{
			"Stacked",
			errs.Pack(errs.New("error1"), errs.New("error2"), errs.New("error3")),
			[]string{"error1", "error2", "error3"},
		},
		{
			"Stacked and Wrapped",
			errs.Pack(errs.New("error1"), errs.Wrap(errs.New("error2"), "error2-wrap"), errs.New("error3")),
			[]string{"error1", "error2-wrap", "error2", "error3"},
		},
		{
			"Stacked, Wrapped and Stacked",
			errs.Pack(
				errs.New("error1"),
				errs.Wrap(
					errs.Pack(errs.New("error2a"), errs.New("error2b")),
					"error2-wrap",
				),
				errs.New("error3")),
			[]string{"error1", "error2-wrap", "error2a", "error2b", "error3"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := errs.Unpack(tt.err)
			errorMsgs := []string{}
			for _, err := range errors {
				errorMsgs = append(errorMsgs, err.Error())
			}
			assert.Equalf(t, tt.want, errorMsgs, "Unpack(%v)", tt.err)
		})
	}
}

func TestJoinMessage(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			"Single",
			errs.New("error1"),
			"error1",
		},
		{
			"Wrapped",
			errs.Wrap(errs.New("error1"), "error2"),
			"error2: error1",
		},
		{
			"Stacked",
			errs.Pack(errs.New("error1"), errs.New("error2"), errs.New("error3")),
			"- error1\n- error2\n- error3",
		},
		{
			"Stacked and Wrapped",
			errs.Pack(
				errs.New("error1"),
				errs.Wrap(errs.New("error2"), "error2-wrap"),
				errs.New("error3"),
			),
			"- error1\n- error2-wrap: error2\n- error3",
		},
		{
			"Stacked, Wrapped and Stacked",
			errs.Pack(
				errs.New("error1"),
				errs.Wrap(
					errs.Pack(errs.New("error2a"), errs.New("error2b")),
					"error2-wrap",
				),
				errs.New("error3")),
			"- error1\n- error2-wrap:\n    - error2a\n    - error2b\n- error3",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := errs.JoinMessage(tt.err)
			assert.Equalf(t, tt.want, msg, "JoinMessage(%v)", tt.err)
		})
	}
}
