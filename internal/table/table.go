package table

import (
	"reflect"

	"github.com/ActiveState/cli/internal/output"
)

type Table struct {
	Info        string      `opts:"hideKey"`
	Rows        interface{} `opts:"hideKey"`
	emptyOutput string
}

func (t *Table) MarshalOutput(format output.Format) interface{} {
	if format == output.PlainFormatName {
		value := reflect.ValueOf(t.Rows)
		if value.Kind() == reflect.Slice && value.Len() == 0 {
			return t.emptyOutput
		}
		return t
	}

	return t.Rows
}

func NewTable(rows interface{}, info, emptyOutput string) *Table {
	return &Table{
		Info:        info,
		Rows:        rows,
		emptyOutput: emptyOutput,
	}
}
