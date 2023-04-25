package packages

import (
	"sort"
	"strings"

	"github.com/ActiveState/cli/internal/output"
)

type tableOutput struct {
	rows        []packageRow
	emptyOutput string // string returned when table is empty
}

type packageRow struct {
	Pkg     string `json:"package" locale:"package_name,Name"`
	Version string `json:"version" locale:"package_version,Version"`
}

func (o *tableOutput) MarshalOutput(format output.Format) interface{} {
	if len(o.rows) == 0 {
		return o.emptyOutput
	}
	return o.rows
}

func (o *tableOutput) MarshalStructured(format output.Format) interface{} {
	return o.rows
}

func newTableOutput(rows []packageRow, emptyOutput string) *tableOutput {
	return &tableOutput{
		rows:        rows,
		emptyOutput: emptyOutput,
	}
}

func (o *tableOutput) sortByPkg() {
	less := func(i, j int) bool {
		a := o.rows[i].Pkg
		b := o.rows[j].Pkg

		if strings.ToLower(a) < strings.ToLower(b) {
			return true
		}

		return a < b
	}

	sort.Slice(o.rows, less)
}
