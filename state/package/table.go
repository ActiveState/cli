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
		if strings.ToLower(rows[i][0]) < strings.ToLower(rows[j][0]) {
			return true
		}
		return rows[i][0] < rows[j][0]
	}
	sort.Slice(rows, less)
}
