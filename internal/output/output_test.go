package output

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		format      Format
		print       interface{}
		expectedOut string
		expectedErr string
	}{
		{
			"plain",
			FormatPlain,
			"hello",
			"hello",
			"",
		},
		{
			"json",
			FormatJSON,
			"hello",
			"\"hello\"",
			"",
		},
		{
			"editor",
			FormatEditor,
			"hello",
			"\"hello\"",
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outWriter := &bytes.Buffer{}
			errWriter := &bytes.Buffer{}

			cfg := &Config{
				OutWriter:   outWriter,
				ErrWriter:   errWriter,
				Colored:     false,
				Interactive: false,
			}

			outputer, fail := New(tt.format, cfg)
			require.NoError(t, fail.ToError())

			outputer.Print(tt.print)

			assert.Equal(t, tt.expectedOut, outWriter.String(), "Output did not match")
			assert.Equal(t, tt.expectedErr, errWriter.String(), "Errors did not match")
		})
	}
}
