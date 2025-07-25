package output

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimple_Notice(t *testing.T) {
	type args struct {
		value interface{}
	}
	tests := []struct {
		name           string
		args           args
		expectedOut    string
		expectedNotice string
	}{
		{
			"Notice should not produce message",
			args{"hello"},
			"",
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outWriter := &bytes.Buffer{}
			errWriter := &bytes.Buffer{}

			f, err := NewSimple(&Config{
				OutWriter:   outWriter,
				ErrWriter:   errWriter,
				Colored:     false,
				Interactive: false,
			})
			require.NoError(t, err)

			f.Notice(tt.args.value)
			assert.Equal(t, tt.expectedOut, outWriter.String(), "Output did not match")
			assert.Equal(t, tt.expectedNotice, errWriter.String(), "Notice did not match")
		})
	}
}
