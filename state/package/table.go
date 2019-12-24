package pkg

import (
	"sort"
	"strings"

	"github.com/bndr/gotabulate"
)

type table struct {
	headers []string
	data    [][]string
}

func newTable(headers []string, data [][]string) *table {
	return &table{
		headers: headers,
		data:    data,
	}
}

func (t *table) output() string {
	if t == nil {
		return ""
	}

	tab := gotabulate.Create(t.data)
	tab.SetHeaders(t.headers)
	tab.SetAlign("left")

	return tab.Render("simple")
}

func sortByFirstCol(rows [][]string) {
	less := func(i, j int) bool {
		if len(rows[i]) == 0 {
			return true
		}
		if len(rows[j]) == 0 {
			return false
		}

		a := rows[i][0]
		b := rows[j][0]

		if strings.ToLower(a) < strings.ToLower(b) {
			return true
		}

		return a < b
	}

	sort.Slice(rows, less)
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
