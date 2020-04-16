package errs_test

import (
	"errors"
	"testing"

	"github.com/ActiveState/cli/internal/errs"
)

func TestNew(t *testing.T) {
	type args struct {
		message string
		args    []interface{}
	}
	tests := []struct {
		name        string
		args        args
		wantErr     bool
		wantMessage string
	}{
		{
			"Creates error, no args",
			args{
				"Error Message",
				[]interface{}{},
			},
			true,
			"Error Message",
		},
		{
			"Creates error, with args",
			args{
				"Error Message %d %s",
				[]interface{}{1, "string"},
			},
			true,
			"Error Message 1 string",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errs.New(tt.args.message, tt.args.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && err.Error() != tt.wantMessage {
				t.Errorf("New() error message = %s, wantMessage %s", err.Error(), tt.wantMessage)
			}
		})
	}
}

type TestError struct{ error }

func TestNewWrapped(t *testing.T) {
	type args struct {
		err     error
		message string
		args    []interface{}
	}
	tests := []struct {
		name        string
		args        args
		wantErr     bool
		wantMessage string
	}{
		{
			"Creates wrapped error, no args",
			args{
				&TestError{errors.New("Original Error Message")},
				"Error Message",
				[]interface{}{},
			},
			true,
			"Error Message",
		},
		{
			"Creates error, with args",
			args{
				&TestError{errors.New("Original Error Message")},
				"Error Message %d %s",
				[]interface{}{1, "string"},
			},
			true,
			"Error Message 1 string",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errs.NewWrapped(tt.args.err, tt.args.message, tt.args.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewWrapped() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil {
				return
			}
			if err.Error() != tt.wantMessage {
				t.Errorf("NewWrapped() error message = %s, wantMessage %s", err.Error(), tt.wantMessage)
			}
			if ! errors.Is(err, tt.args.err) {
				t.Errorf("error should wrap provided error")
			}
		})
	}
}

func TestNewError(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		wantLog bool
	}{
		{
			"Creates error",
			args{
				errors.New("Error Message"),
			},
			true,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logCalled := false
			err := errs.NewErrorWithLogger(tt.args.err, func(string, ...interface{}) { logCalled = true })
			if (err != nil) != tt.wantErr {
				t.Errorf("NewError() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.wantErr && err != tt.args.err {
				t.Error("NewError() error pointer is not the same as the passed error")
			}
			if tt.wantLog != logCalled {
				t.Errorf("NewError() logCalled = %v, wantLog %v", logCalled, tt.wantLog)
			}
		})
	}
}

func TestLocalize(t *testing.T) {
	type args struct {
		message string
		locale  string
	}
	tests := []struct {
		name        string
		args        args
		wantErr     bool
		wantMessage string
	}{
		{
			"Creates localized error",
			args{
				"Original Error Message",
				"Localized Error",
			},
			true,
			"Localized Error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errOrig := errs.New(tt.args.message)
			err := errs.Localize(errOrig, tt.args.locale)
			if (err != nil) != tt.wantErr {
				t.Errorf("Localize() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && err.Error() != tt.wantMessage {
				t.Errorf("Localize() error message = %s, wantMessage %s", err.Error(), tt.wantMessage)
			}
			var errLocale errs.Localizer = &errs.LocalizedError{}
			if ok := errors.As(err, &errLocale); !ok {
				t.Error("Error should be compatible with Localizer interface")
			}
			if ok := errs.IsLocale(err); !ok {
				t.Error("IsLocale cannot determine that error is localized")
			}
		})
	}
}
