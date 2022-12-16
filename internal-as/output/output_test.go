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
		formatName  string
		print       interface{}
		expectedOut string
		expectedErr string
	}{
		{
			"plain",
			"plain",
			"hello",
			"hello\n",
			"",
		},
		{
			"json",
			"json",
			"hello",
			`"hello"` + "\x00\n",
			"",
		},
		{
			"editor",
			"editor",
			"hello",
			`"hello"` + "\x00\n",
			"",
		},
		{
			"editor.v0",
			"editor.v0",
			"hello",
			`"hello"` + "\n",
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

			outputer, err := New(tt.formatName, cfg)
			require.NoError(t, err)

			outputer.Print(tt.print)

			assert.Equal(t, tt.expectedOut, outWriter.String(), "Output did not match")
			assert.Equal(t, tt.expectedErr, errWriter.String(), "Errors did not match")
		})
	}
}
