package table

import (
	"strings"

	"github.com/ActiveState/cli/internal/sliceutils"
	"github.com/ActiveState/cli/internal/termutils"
)

const dash = "\u2500"
const linebreak = "\n"
const padding = 2

type FormatFunc func(string, ...interface{}) string

type row struct {
	columns []string
}

type Table struct {
	headers []string
	rows    []row
}

func New(headers []string) *Table {
	return &Table{headers, []row{}}
}

func (t *Table) AddRow(vs ...[]string) *Table {
	for _, v := range vs {
		t.rows = append(t.rows, row{v})
	}
	return t
}

func (t *Table) Render() string {
	if len(t.rows) == 0 {
		return ""
	}

	termWidth := termutils.GetWidth()
	colWidths, total := t.calculateWidth(termWidth)

	out := ""
	out += renderRow(t.headers, colWidths)
	out += "[DISABLED]" + strings.Repeat(dash, total) + "[/RESET]"
	for _, row := range t.rows {
		out += renderRow(row.columns, colWidths)
	}

	return out
}

func (t *Table) calculateWidth(maxTotalWidth int) ([]int, int) {
	// Calculate required width of each column
	colWidths := make([]int, len(t.headers))
	for n, header := range t.headers {
		if currentSize, ok := sliceutils.GetInt(colWidths, n); !ok || currentSize < len(header) {
			colWidths[n] = len(header)
		}
		for _, row := range t.rows {
			if rowValue, ok := sliceutils.GetString(row.columns, n); ok && colWidths[n] < len(rowValue) {
				colWidths[n] = len(row.columns[n])
			}
		}
	}

	// Add padding and calculate total
	total := 0
	for n, w := range colWidths {
		colWidths[n] = w + (padding * 2)
		total += colWidths[n]
	}

	// If over max total; reduce size according to percentage of total
	if total > maxTotalWidth {
		for n, w := range colWidths {
			colWidths[n] = int(float64(w) / float64(total) * float64(maxTotalWidth))
		}
	}

	return colWidths, total
}

func renderRow(providedColumns []string, colWidths []int) string {
	// don't want to modify the provided slice
	columns := make([]string, len(providedColumns))
	copy(columns, providedColumns)

	lastIterLen := -1
	result := ""

	// Keep rendering lines until there's no column data left to render
	for len(result) != lastIterLen {
		lastIterLen = len(result) + len(linebreak)

		// Iterate over the columns by their line sizes
		for n, maxlen := range colWidths {
			if len(columns) < n+1 {
				continue
			}

			colValue := columns[n]
			if colValue == "" {
				break
			}

			// Detect multi column span
			if len(colWidths) > n+1 && len(columns) == n+1 {
				for _, v := range colWidths[n+1:] {
					maxlen += v
				}
			}

			maxlen = maxlen - (padding * 2)

			// How much of the colValue are we using this line?
			end := len(colValue)
			if end > maxlen {
				end = maxlen
			}

			suffix := strings.Repeat(" ", maxlen-end)
			result += pad(colValue[0:end] + suffix)
			columns[n] = colValue[end:]
		}
		result += linebreak
	}

	return strings.TrimRight(result, linebreak)
}

func pad(v string) string {
	padded := strings.Repeat(" ", padding)
	return padded + v + padded
}
