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
			[]int{padsize(3), padsize(4), padsize(5)},
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
			[]int{padsize(3), padsize(4), padsize(5)},
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
			[]int{padsize(23), padsize(29), padsize(35)},
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
			"Mutli column span",
			args{
				providedColumns: []string{"col1", "col2"},
				colWidths:       []int{6, 6, 6},
			},
			"  co    col2      \n" +
				"  l1  ",
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
