//go:build !windows
// +build !windows

// Can't test this on Windows since on Windows it sends process instructions to change colors

package colorize

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_writeColorized(t *testing.T) {
	tests := []struct {
		name     string
		strip    bool
		value    string
		expected string
	}{
		{
			`heading`,
			false,
			`heading: [HEADING]value[/RESET] -- end`,
			"heading: \x1b[39;1m\x1b[1mvalue\x1b[0m -- end",
		},
		{
			`notice`,
			false,
			`notice: [NOTICE]value[/RESET] -- end`,
			"notice: \x1b[39;1mvalue\x1b[0m -- end",
		},
		{
			`error`,
			false,
			`error: [ERROR]value[/RESET] -- end`,
			"error: \x1b[31mvalue\x1b[0m -- end",
		},
		{
			`disabled`,
			false,
			`disabled: [DISABLED]value[/RESET] -- end`,
			"disabled: \x1b[2mvalue\x1b[0m -- end",
		},
		{
			`highlight`,
			false,
			`highlight: [ACTIONABLE]value[/RESET] -- end`,
			"highlight: \x1b[36;1mvalue\x1b[0m -- end",
		},
		{
			`stripped`,
			true,
			`white: [ERROR]value[/RESET] [ACTIONABLE]highlighted value[/RESET] -- end`,
			"white: value highlighted value -- end",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := &bytes.Buffer{}
			_, err := Colorize(tt.value, writer, tt.strip)
			assert.NoError(t, err, "Colorize failed")
			assert.Equal(t, tt.expected, writer.String(), "Output did not match")
		})
	}
}
