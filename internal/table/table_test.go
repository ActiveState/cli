package table

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTable_colWidths(t1 *testing.T) {
	type args struct {
		table         *Table
		maxTotalWidth int
	}
	tests := []struct {
		name string
		args args
		want []int
	}{
		{
			"Under max total",
			args{
				&Table{
					[]string{"123", "1234", "12345"},
					[]row{
						{[]string{"1", "2", "3"}},
					},
				},
				100,
			},
			[]int{8, 9, 10},
		},
		{
			"multi-byte characters",
			args{
				&Table{
					[]string{"12✔", "1234", "12345"},
					[]row{
						{[]string{"1", "2", "3"}},
					},
				},
				100,
			},
			[]int{8, 9, 10},
		},
		{
			"span row dominates",
			args{
				&Table{
					[]string{"123", "1234", "12345"},
					[]row{
						{[]string{"1", "2", "3"}},
						{[]string{"1", "0123456789012345678901234567890123456789"}},
					},
				},
				100,
			},
			[]int{15, 17, 19},
		},
		{
			"Rowsize wins cause it's longer",
			args{
				&Table{
					[]string{"1", "2", "3"},
					[]row{
						{[]string{"123", "1234", "12345"}},
					},
				},
				100,
			},
			[]int{8, 9, 10},
		},
		{
			"Over max total",
			args{
				&Table{
					[]string{strings.Repeat("-", 40), strings.Repeat("-", 50), strings.Repeat("-", 60)},
					[]row{
						{[]string{"1", "2", "3"}},
					},
				},
				100,
			},
			[]int{28, 33, 39},
		},
		{
			"Long multi column",
			args{
				&Table{
					[]string{"a", "b", "c", "d"},
					[]row{
						{[]string{"1", "1", "12", "12"}},
						{[]string{strings.Repeat(" ", 100)}},
					},
				},
				100,
			},
			[]int{23, 23, 27, 27},
		},
		{
			"Long multi column (over maxWidth)",
			args{
				&Table{
					[]string{"a", "b", "c", "d"},
					[]row{
						{[]string{"1", "1", "12", "12"}},
						{[]string{"1", strings.Repeat(" ", 200)}},
					},
				},
				100,
			},
			[]int{23, 23, 27, 27},
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			got, total := tt.args.table.calculateWidth(tt.args.maxTotalWidth)
			if !reflect.DeepEqual(got, tt.want) {
				t1.Errorf("calculateWidth() = %v, want %v", got, tt.want)
			}

			assert.LessOrEqual(t1, total, tt.args.maxTotalWidth)
		})
	}
}

func padsize(v int) int {
	return v + (2 * padding)
}

func Test_renderRow(t *testing.T) {
	type args struct {
		providedColumns []string
		colWidths       []int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"No breaks",
			args{
				providedColumns: []string{"col1", "col2", "col3"},
				colWidths:       []int{10, 10, 10},
			},
			"  col1      col2      col3    ",
		},
		{
			"Breaks",
			args{
				providedColumns: []string{"col1", "col2", "col3"},
				colWidths:       []int{6, 6, 6},
			},
			"  co    co    co  \n" +
				"  l1    l2    l3  ",
		},
		{
			"Breaks for multi-byte characters",
			args{
				providedColumns: []string{"✔ol1", "✔ol2", "✔ol3"},
				colWidths:       []int{6, 6, 6},
			},
			"  ✔o    ✔o    ✔o  \n" +
				"  l1    l2    l3  ",
		},
		{
			"Empty column",
			args{
				providedColumns: []string{"col1", "", "col3"},
				colWidths:       []int{6, 6, 6},
			},
			"  co          co  \n" +
				"  l1          l3  ",
		},
		{
			"Mutli column span",
			args{
				providedColumns: []string{"abcdefgh", "jklmnopqrstu"},
				colWidths:       []int{6, 6, 6},
			},
			"  ab    jklmnopq  \n" +
				"  cd    rstu      \n" +
				"  ef              \n" +
				"  gh              ",
		},
		{
			"Single row mutli column span",
			args{
				providedColumns: []string{"123456789"},
				colWidths:       []int{1, 2, 3, 4, 5},
			},
			"  123456789    ",
		},
		{
			"Multi line second column",
			args{
				providedColumns: []string{"abcd", "abcdefgh"},
				colWidths:       []int{8, 8},
			},
			"  abcd    abcd  \n" +
				"          efgh  ",
		},
		{
			"Multi line second column with line breaks",
			args{
				providedColumns: []string{"abcd", "abcde\nfgh"},
				colWidths:       []int{8, 8},
			},
			"  abcd    abcd  \n" +
				"          e    \n" +
				"          fgh   ",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := renderRow(tt.args.providedColumns, tt.args.colWidths); got != tt.want {
				t.Errorf("renderRow() = '%v', want '%v'", renderBreaks(got), renderBreaks(tt.want))
			}
		})
	}
}

func renderBreaks(v string) string {
	return strings.ReplaceAll(v, "\n", `\n`)
}

func Test_getCroppedText(t *testing.T) {
	type args struct {
		text   string
		maxLen int
	}
	tests := []struct {
		name string
		args args
		want []entry
	}{
		{
			"No split",
			args{"[HEADING]Hello[/RESET]", 5},
			[]entry{{"[HEADING]Hello[/RESET]", 5}},
		},
		{
			"Split",
			args{"[HEADING]Hello[/RESET]", 3},
			[]entry{{"[HEADING]Hel", 3}, {"lo[/RESET]", 2}},
		},
		{
			"Split multiple",
			args{"[HEADING]Hello World[/RESET]", 3},
			[]entry{{"[HEADING]Hel", 3}, {"lo ", 3}, {"Wor", 3}, {"ld[/RESET]", 2}},
		},
		{
			"Split multiple no match",
			args{"Hello World", 3},
			[]entry{{"Hel", 3}, {"lo ", 3}, {"Wor", 3}, {"ld", 2}},
		},
		{
			"No split no match",
			args{"Hello", 5},
			[]entry{{"Hello", 5}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getCroppedText(tt.args.text, tt.args.maxLen); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getCroppedText() = %v, want %v", got, tt.want)
			}
		})
	}
}
