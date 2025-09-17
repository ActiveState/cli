package colorize

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Wrap(t *testing.T) {
	type args struct {
		text   string
		maxLen int
	}
	tests := []struct {
		name string
		args args
		want WrappedLines
	}{
		{
			"No split",
			args{"[HEADING]Hello[/RESET]", 5},
			[]WrappedLine{{"[HEADING]Hello[/RESET]", 5}},
		},
		{
			"Split",
			args{"[HEADING]Hello[/RESET]", 3},
			[]WrappedLine{{"[HEADING]Hel", 3}, {"lo[/RESET]", 2}},
		},
		{
			"Split multiple",
			args{"[HEADING]Hello World[/RESET]", 3},
			[]WrappedLine{{"[HEADING]Hel", 3}, {"lo ", 3}, {"Wor", 3}, {"ld[/RESET]", 2}},
		},
		{
			"Split multiple no match",
			args{"Hello World", 3},
			[]WrappedLine{{"Hel", 3}, {"lo ", 3}, {"Wor", 3}, {"ld", 2}},
		},
		{
			"No split no match",
			args{"Hello", 5},
			[]WrappedLine{{"Hello", 5}},
		},
		{
			"Split multi-byte characters",
			args{"✔ol1✔ol2✔ol3", 4},
			[]WrappedLine{{"✔ol1", 4}, {"✔ol2", 4}, {"✔ol3", 4}},
		},
		{
			"No split multi-byte character with tags",
			args{"[HEADING]✔ Some Text[/RESET]", 20},
			[]WrappedLine{{"[HEADING]✔ Some Text[/RESET]", 11}},
		},
		{
			"Split multi-byte character with tags",
			args{"[HEADING]✔ Some Text[/RESET]", 6},
			[]WrappedLine{{"[HEADING]✔ Some", 6}, {" Text[/RESET]", 5}},
		},
		{
			"Split multi-byte character with tags by words",
			args{"[HEADING]✔ Some Text[/RESET]", 10},
			[]WrappedLine{{"[HEADING]✔ Some ", 7}, {"Text[/RESET]", 4}},
		},
		{
			"Split line break",
			args{"[HEADING]Hel\nlo[/RESET]", 5},
			[]WrappedLine{{"[HEADING]Hel", 3}, {"lo[/RESET]", 2}},
		},
		{
			"Split nested",
			args{"[HEADING][NOTICE]Hello[/RESET][/RESET]", 3},
			[]WrappedLine{{"[HEADING][NOTICE]Hel", 3}, {"lo[/RESET][/RESET]", 2}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Wrap(tt.args.text, tt.args.maxLen, false, ""); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Wrap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_WrapAsString(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{
			// This should only give two empty lines, because the first line break is effectively part of the first line
			"Line Endings",
			"test\n\n\ntest",
		},
		{
			"Ends with Multiple Line Endings",
			"test\n\n\n",
		},
		{
			"Starts with Multiple Line Endings",
			"\n\n\ntest",
		},
		{
			"Double Line Ending",
			"X\n\n",
		},
		{
			"Double Line Endings",
			"X\n\nX\n\nX\n\n",
		},
		{
			"Just Line Endings",
			"\n\n\n",
		},
		{
			"Single Line Ending",
			"\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Wrap(tt.text, 999, true, ""); !reflect.DeepEqual(got.String(), tt.text) {
				escape := func(v string) string {
					return strings.Replace(v, "\n", "\\n", -1)
				}
				t.Errorf("Wrap() = %v, want %v (wrap data: %s)", escape(got.String()), escape(tt.text), escape(fmt.Sprintf("%#v", got)))
			}
		})
	}
}

func TestWrapBullet(t *testing.T) {
	lines := Wrap(" • This is a bullet", 15, true, "")
	assert.Equal(t, " • This is a \n   bullet", lines.String())
}

func TestWrapContinuation(t *testing.T) {
	// Test normal wrapping with no continuation.
	lines := Wrap("This is an error", 9, true, "")
	assert.Equal(t, "This is \nan error", lines.String())

	// Verify continuations are not tagged.
	lines = Wrap("[ERROR]This is an error[/RESET]", 10, true, "|")
	assert.Equal(t, "[ERROR]This is \n[/RESET]|[ERROR]an error[/RESET]", lines.String())

	// Verify only active tags come after continuations.
	lines = Wrap("[BOLD]This is not[/RESET] an error", 10, true, "|")
	assert.Equal(t, "[BOLD]This is \n[/RESET]|[BOLD]not[/RESET] an \n|error", lines.String())

	// Verify continuations are not tagged, even if [/RESET] is omitted.
	lines = Wrap("[ERROR]This is an error", 10, true, "|")
	assert.Equal(t, "[ERROR]This is \n[/RESET]|[ERROR]an error", lines.String())

	// Verify multiple tags are restored after continuations.
	lines = Wrap("[BOLD][RED]This is a bold, red message[/RESET]", 11, true, "|")
	assert.Equal(t, "[BOLD][RED]This is a \n[/RESET]|[BOLD][RED]bold, red \n[/RESET]|[BOLD][RED]message[/RESET]", lines.String())
}
