package output

import (
	"bytes"
	"errors"
	"testing"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSON_Print(t *testing.T) {
	type args struct {
		value interface{}
	}
	tests := []struct {
		name        string
		args        args
		expectedOut string
		expectedErr string
	}{
		{
			"simple string",
			args{"hello"},
			`"hello"`,
			"",
		},
		{
			"error string",
			args{errors.New("hello")},
			`"hello"`,
			"",
		},
		{
			"struct",
			args{
				struct {
					Field1 string
					Field2 string
					field3 string
				}{
					"value1", "value2", "value3",
				},
			},
			`{"Field1":"value1","Field2":"value2"}`,
			"",
		},
		{
			"struct with json tags",
			args{
				struct {
					Field1 string `json:"RealField1"`
					Field2 string `json:"RealField2"`
					field3 string
				}{
					"value1", "value2", "value3",
				},
			},
			`{"RealField1":"value1","RealField2":"value2"}`,
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outWriter := &bytes.Buffer{}
			errWriter := &bytes.Buffer{}

			f, err := NewJSON(&Config{
				OutWriter:   outWriter,
				ErrWriter:   errWriter,
				Colored:     false,
				Interactive: false,
			})
			require.NoError(t, err)

			f.Print(tt.args.value)
			assert.Equal(t, tt.expectedOut, outWriter.String(), "Output did not match")
			assert.Equal(t, tt.expectedErr, errWriter.String(), "Errors did not match")
		})
	}
}

func TestJSON_Notice(t *testing.T) {
	type args struct {
		value interface{}
	}
	tests := []struct {
		name        string
		args        args
		expectedOut string
		expectedErr string
	}{
		{
			"simple string",
			args{"hello"},
			"",
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outWriter := &bytes.Buffer{}
			errWriter := &bytes.Buffer{}

			f, err := NewJSON(&Config{
				OutWriter:   outWriter,
				ErrWriter:   errWriter,
				Colored:     false,
				Interactive: false,
			})
			require.NoError(t, err)

			f.Notice(tt.args.value)
			assert.Equal(t, tt.expectedOut, outWriter.String(), "Output did not match")
			assert.Equal(t, tt.expectedErr, errWriter.String(), "Errors did not match")
		})
	}
}

func TestJSON_Error(t *testing.T) {
	type args struct {
		value interface{}
	}
	tests := []struct {
		name        string
		args        args
		expectedOut string
		expectedErr string
	}{
		{
			"localized error",
			args{locale.NewError("", "hello")},
			`{"error":"hello"}`,
			"",
		},
		{
			"simple string",
			args{"hello"},
			`{"error":"hello"}`,
			"",
		},
		{
			"unrecognized",
			args{1},
			`{"error":"Not a recognized error format: 1"}`,
			"",
		},
		{
			"raw JSON",
			args{[]byte(`"hello"`)},
			`"hello"`,
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outWriter := &bytes.Buffer{}
			errWriter := &bytes.Buffer{}

			f, err := NewJSON(&Config{
				OutWriter:   outWriter,
				ErrWriter:   errWriter,
				Colored:     false,
				Interactive: false,
			})
			require.NoError(t, err)

			f.Error(tt.args.value)
			assert.Equal(t, tt.expectedOut, outWriter.String(), "Output did not match")
			assert.Equal(t, tt.expectedErr, errWriter.String(), "Errors did not match")
		})
	}
}
