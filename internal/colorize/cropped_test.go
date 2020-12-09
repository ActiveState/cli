package colorize

import (
	"reflect"
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
			"Split line break",
			args{"[HEADING]Hel\nlo[/RESET]", 5},
			[]CroppedLine{{"[HEADING]Hel", 3}, {"lo[/RESET]", 2}},
		},
		{
			"Split nested",
			args{"[HEADING][NOTICE]Hello[/RESET][/RESET]", 3},
			[]CroppedLine{{"[HEADING][NOTICE]Hel", 3}, {"lo[/RESET][/RESET]", 2}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetCroppedText([]rune(tt.args.text), tt.args.maxLen); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getCroppedText() = %v, want %v", got, tt.want)
			}
		})
	}
}
