package packages

import (
	"sort"
	"strings"
)

type packageTable struct {
	Info        string       `locale:"info" opts:"hideKey"`
	Rows        []packageRow `locale:"rows" opts:"hideKey"`
	emptyOutput string       // string returned when table is empty
}

type packageRow struct {
	Pkg     string `json:"package" locale:"package_name,Name"`
	Version string `json:"version" locale:"package_version,Version"`
}

func sortByPkg(rows []packageRow) {
	less := func(i, j int) bool {
		a := rows[i].Pkg
		b := rows[j].Pkg

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
