package table

import (
	"reflect"
	"strings"
	"testing"
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
						row{[]string{"1", "2", "3"}},
					},
				},
				100,
			},
			[]int{13, 14, 15},
		},
		{
			"Rowsize wins cause it's longer",
			args{
				&Table{
					[]string{"1", "2", "3"},
					[]row{
						row{[]string{"123", "1234", "12345"}},
					},
				},
				100,
			},
			[]int{13, 14, 15},
		},
		{
			"Over max total",
			args{
				&Table{
					[]string{strings.Repeat("-", 40), strings.Repeat("-", 50), strings.Repeat("-", 60)},
					[]row{
						row{[]string{"1", "2", "3"}},
					},
				},
				100,
			},
			[]int{27, 33, 40},
		},
		{
			"Long multi column",
			args{
				&Table{
					[]string{"a", "b", "c", "d"},
					[]row{
						row{[]string{"1", "1", "12", "12"}},
						row{[]string{strings.Repeat(" ", 100)}},
					},
				},
				100,
			},
			[]int{23, 23, 26, 28},
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			if got, _ := tt.args.table.calculateWidth(tt.args.maxTotalWidth); !reflect.DeepEqual(got, tt.want) {
				t1.Errorf("calculateWidth() = %v, want %v", got, tt.want)
			}
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
				"          e\n    \n" +
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
