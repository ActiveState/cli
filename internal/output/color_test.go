// +build !windows
// Can't test this on Windows since on Windows it sends process instructions to change colors

package output

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_writeColorized(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{
			`bold`,
			`bold: [BOLD]value[/RESET] -- end`,
			"bold: \x1b[1mvalue\x1b[0m -- end",
		},
		{
			`underline`,
			`underline: [UNDERLINE]value[/RESET] -- end`,
			"underline: \x1b[4mvalue\x1b[0m -- end",
		},
		{
			`black`,
			`black: [BLACK]value[/RESET] [BLACK!]bright value[/RESET] -- end`,
			"black: \x1b[0;30mvalue\x1b[0m \x1b[0;30;1mbright value\x1b[0m -- end",
		},
		{
			`red`,
			`red: [RED]value[/RESET] [RED!]bright value[/RESET] -- end`,
			"red: \x1b[0;31mvalue\x1b[0m \x1b[0;31;1mbright value\x1b[0m -- end",
		},
		{
			`green`,
			`green: [GREEN]value[/RESET] [GREEN!]bright value[/RESET] -- end`,
			"green: \x1b[0;32mvalue\x1b[0m \x1b[0;32;1mbright value\x1b[0m -- end",
		},
		{
			`yellow`,
			`yellow: [YELLOW]value[/RESET] [YELLOW!]bright value[/RESET] -- end`,
			"yellow: \x1b[0;33mvalue\x1b[0m \x1b[0;33;1mbright value\x1b[0m -- end",
		},
		{
			`blue`,
			`blue: [BLUE]value[/RESET] [BLUE!]bright value[/RESET] -- end`,
			"blue: \x1b[0;34mvalue\x1b[0m \x1b[0;34;1mbright value\x1b[0m -- end",
		},
		{
			`magenta`,
			`magenta: [MAGENTA]value[/RESET] [MAGENTA!]bright value[/RESET] -- end`,
			"magenta: \x1b[0;35mvalue\x1b[0m \x1b[0;35;1mbright value\x1b[0m -- end",
		},
		{
			`cyan`,
			`cyan: [CYAN]value[/RESET] [CYAN!]bright value[/RESET] -- end`,
			"cyan: \x1b[0;36mvalue\x1b[0m \x1b[0;36;1mbright value\x1b[0m -- end",
		},
		{
			`white`,
			`white: [WHITE]value[/RESET] [WHITE!]bright value[/RESET] -- end`,
			"white: \x1b[0;37mvalue\x1b[0m \x1b[0;37;1mbright value\x1b[0m -- end",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := &bytes.Buffer{}
			writeColorized(tt.value, writer)
			assert.Equal(t, tt.expected, writer.String(), "Output did not match")
		})
	}
}
