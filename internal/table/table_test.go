package table

import (
	"reflect"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/mathutils"
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
					false,
					false,
				},
				100,
			},
			[]int{7, 8, 9},
		},
		{
			"multi-byte characters",
			args{
				&Table{
					[]string{"12✔", "1234", "12345"},
					[]row{
						{[]string{"1", "2", "3"}},
					},
					false,
					false,
				},
				100,
			},
			[]int{7, 8, 9},
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
					false,
					false,
				},
				100,
			},
			[]int{14, 17, 20},
		},
		{
			"Rowsize wins cause it's longer",
			args{
				&Table{
					[]string{"1", "2", "3"},
					[]row{
						{[]string{"123", "1234", "12345"}},
					},
					false,
					false,
				},
				100,
			},
			[]int{7, 8, 9},
		},
		{
			"Over max total",
			args{
				&Table{
					[]string{strings.Repeat("-", 40), strings.Repeat("-", 50), strings.Repeat("-", 60)},
					[]row{
						{[]string{"1", "2", "3"}},
					},
					false,
					false,
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
					false,
					false,
				},
				100,
			},
			[]int{22, 22, 27, 29},
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
					false,
					false,
				},
				100,
			},
			[]int{22, 22, 27, 29},
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
			"No breaks with color codes",
			args{
				providedColumns: []string{"[HEADING]col1[/RESET]", "[HEADING]col2[/RESET]", "[HEADING]col3[/RESET]"},
				colWidths:       []int{10, 10, 10},
			},
			"  [HEADING]col1[/RESET]      [HEADING]col2[/RESET]      [HEADING]col3[/RESET]    ",
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
			"Breaks with color codes",
			args{
				providedColumns: []string{"[HEADING]col1[/RESET]", "[HEADING]col2[/RESET]", "[HEADING]col3[/RESET]"},
				colWidths:       []int{6, 6, 6},
			},
			"  [HEADING]co    [HEADING]co    [HEADING]co  \n" +
				"  l1[/RESET]    l2[/RESET]    l3[/RESET]  ",
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
				"          e     \n" +
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

func Test_equalizeWidths(t *testing.T) {
	type args struct {
		colWidths  []int
		percentage int
	}
	tests := []struct {
		name string
		args args
		want []int
	}{
		{
			"Equalize widths",
			args{
				[]int{10, 20, 30},
				20,
			},
			[]int{12, 20, 28},
		},
		{
			"Equalize widths, account for floats",
			args{
				[]int{1, 1, 5},
				40,
			},
			[]int{1, 1, 5},
		},
		{
			"Zero percentage doesn't panic or break",
			args{
				[]int{11, 21, 31},
				0,
			},
			[]int{11, 21, 31},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalTotal := mathutils.Total(tt.args.colWidths...)
			equalizeWidths(tt.args.colWidths, tt.args.percentage)
			if !reflect.DeepEqual(tt.args.colWidths, tt.want) {
				t.Errorf("equalizeWidths() got = %v, want %v", tt.args.colWidths, tt.want)
			}
			if originalTotal != mathutils.Total(tt.args.colWidths...) {
				t.Errorf("Output total should be equal to input total, got: %v", tt.args.colWidths)
			}
		})
	}
}

func Test_rescaleColumns(t *testing.T) {
	type args struct {
		colWidths   []int
		targetTotal int
	}
	tests := []struct {
		name string
		args args
		want []int
	}{
		{
			"Rescale widths, bigger",
			args{
				[]int{5, 5, 5},
				20,
			},
			[]int{6, 6, 8},
		},
		{
			"Rescale widths, same",
			args{
				[]int{5, 5, 5},
				15,
			},
			[]int{5, 5, 5},
		},
		{
			"Rescale widths, smaller",
			args{
				[]int{5, 5, 5},
				10,
			},
			[]int{3, 3, 4},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rescaleColumns(tt.args.colWidths, tt.args.targetTotal, false, tt.args.colWidths[0])
			if !reflect.DeepEqual(tt.args.colWidths, tt.want) {
				t.Errorf("rescaleColumns() got = %v, want %v", tt.args.colWidths, tt.want)
			}
			total := mathutils.Total(tt.args.colWidths...)
			if tt.args.targetTotal != total {
				t.Errorf("rescaleColumns() got total = %v, want total %v", total, tt.args.targetTotal)
			}
		})
	}
}
