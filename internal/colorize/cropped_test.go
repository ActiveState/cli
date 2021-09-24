package colorize

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func Test_GetCroppedText(t *testing.T) {
	type args struct {
		text   string
		maxLen int
	}
	tests := []struct {
		name string
		args args
		want CroppedLines
	}{
		{
			"No split",
			args{"[HEADING]Hello[/RESET]", 5},
			[]CroppedLine{{"[HEADING]Hello[/RESET]", 5}},
		},
		{
			"Split",
			args{"[HEADING]Hello[/RESET]", 3},
			[]CroppedLine{{"[HEADING]Hel", 3}, {"lo[/RESET]", 2}},
		},
		{
			"Split multiple",
			args{"[HEADING]Hello World[/RESET]", 3},
			[]CroppedLine{{"[HEADING]Hel", 3}, {"lo ", 3}, {"Wor", 3}, {"ld[/RESET]", 2}},
		},
		{
			"Split multiple no match",
			args{"Hello World", 3},
			[]CroppedLine{{"Hel", 3}, {"lo ", 3}, {"Wor", 3}, {"ld", 2}},
		},
		{
			"No split no match",
			args{"Hello", 5},
			[]CroppedLine{{"Hello", 5}},
		},
		{
			"Split multi-byte characters",
			args{"✔ol1✔ol2✔ol3", 4},
			[]CroppedLine{{"✔ol1", 4}, {"✔ol2", 4}, {"✔ol3", 4}},
		},
		{
			"No split multi-byte character with tags",
			args{"[HEADING]✔ Some Text[/RESET]", 20},
			[]CroppedLine{{"[HEADING]✔ Some Text[/RESET]", 11}},
		},
		{
			"Split multi-byte character with tags",
			args{"[HEADING]✔ Some Text[/RESET]", 6},
			[]CroppedLine{{"[HEADING]✔ Some", 6}, {" Text[/RESET]", 5}},
		},
		{
			"Split line break",
			args{"[HEADING]Hel\nlo[/RESET]", 5},
			[]CroppedLine{{"[HEADING]Hel", 3}, {"lo[/RESET]", 2}},
		},
		{
			"Split nested",
			args{"[HEADING][NOTICE]Hello[/RESET][/RESET]", 3},
			[]CroppedLine{{"[HEADING][NOTICE]Hel", 3}, {"lo[/RESET][/RESET]", 2}},
		},
		{
			// This should only give two empty lines, because the first line break is effectively part of the first line
			"Line Endings",
			args{"test\n\n\ntest", 4},
			[]CroppedLine{{"test", 4}, {"", 0}, {"", 0}, {"test", 4}},
		},
		{
			"Ends with Multiple Line Endings",
			args{"test\n\n\n", 4},
			[]CroppedLine{{"test", 4}, {"", 0}, {"", 0}},
		},
		{
			"Starts with Multiple Line Endings",
			args{"\n\n\ntest", 4},
			[]CroppedLine{{"", 0}, {"", 0}, {"", 0}, {"test", 4}},
		},
		{
			"Double Line Ending",
			args{"X\n\n", 100},
			[]CroppedLine{{"X", 1}, {"", 0}, {"", 0}},
		},
		{
			"Double Line Endings",
			args{"X\n\nX\n\nX\n\n", 100},
			[]CroppedLine{{"X", 1}, {"", 0}, {"X", 1}, {"", 0}, {"X", 1}, {"", 0}},
		},
		{
			"Just Line Endings",
			args{"\n\n\n", 3},
			[]CroppedLine{{"", 0}, {"", 0}, {"", 0}, {"", 0}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetCroppedText(tt.args.text, tt.args.maxLen, false); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getCroppedText() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_GetCroppedTextAsString(t *testing.T) {
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
			if got := GetCroppedText(tt.text, 999, true); !reflect.DeepEqual(got.String(), tt.text) {
				escape := func(v string) string {
					return strings.Replace(v, "\n", "\\n", -1)
				}
				t.Errorf("getCroppedText() = %v, want %v (crop data: %s)", escape(got.String()), escape(tt.text), escape(fmt.Sprintf("%#v", got)))
			}
		})
	}
}
