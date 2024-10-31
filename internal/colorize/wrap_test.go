package colorize

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
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
				t.Errorf("Wrap() = %v, want %v (crop data: %s)", escape(got.String()), escape(tt.text), escape(fmt.Sprintf("%#v", got)))
			}
		})
	}
}
