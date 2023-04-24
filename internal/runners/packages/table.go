package packages

import (
	"sort"
	"strings"

	"github.com/ActiveState/cli/internal/output"
)

type packageTable struct {
	rows        []packageRow
	emptyOutput string // string returned when table is empty
}

type packageRow struct {
	Pkg     string `json:"package" locale:"package_name,Name"`
	Version string `json:"version" locale:"package_version,Version"`
}

func (t *packageTable) MarshalOutput(format output.Format) interface{} {
	if len(t.rows) == 0 {
		return t.emptyOutput
	}
	return t.rows
}

func (t *packageTable) MarshalStructured(format output.Format) interface{} {
	return t.rows
}

func newTable(rows []packageRow, emptyOutput string) *packageTable {
	return &packageTable{
		rows:        rows,
		emptyOutput: emptyOutput,
	}
}

func (t *packageTable) sortByPkg() {
	less := func(i, j int) bool {
		a := t.rows[i].Pkg
		b := t.rows[j].Pkg

		if strings.ToLower(a) < strings.ToLower(b) {
			return true
		}

		return a < b
	}

	sort.Slice(t.rows, less)
}
