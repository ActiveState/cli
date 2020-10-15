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
	if t == nil {
		return nil
	}

	if format == output.PlainFormatName {
		if len(t.rows) == 0 {
			return t.emptyOutput
		}
		return t.rows
	}

	type packageRow struct {
		Pkg     string `json:"package"`
		Version string `json:"version"`
	}

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

func sortByFirstTwoCols(rows [][]string) {
	less := func(i, j int) bool {
		if len(rows[i]) < 2 {
			return true
		}
		if len(rows[j]) < 2 {
			return false
		}

		aa, ab := rows[i][0], rows[i][1]
		ba, bb := rows[j][0], rows[j][1]

		if strings.ToLower(aa) < strings.ToLower(ba) {
			return true
		}

		if aa > ba {
			return false
		}

		if strings.ToLower(ab) < strings.ToLower(bb) {
			return true
		}

		return ab < bb
	}

	sort.Slice(rows, less)
}
