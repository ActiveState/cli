package table

import (
	"math"
	"strings"

	"github.com/ActiveState/cli/internal/logging"
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
	out += "[NOTICE]" + renderRow(t.headers, colWidths) + "[/RESET]" + linebreak
	out += "[DISABLED]" + strings.Repeat(dash, total) + "[/RESET]" + linebreak
	for _, row := range t.rows {
		out += renderRow(row.columns, colWidths) + linebreak
	}

	return strings.TrimRight(out, linebreak)
}

func (t *Table) calculateWidth(maxTotalWidth int) ([]int, int) {
	// Calculate required width of each column
	minWidth := padding * 2
	colWidths := make([]int, len(t.headers))
	for n, header := range t.headers {
		// Check header sizes
		if currentSize, ok := sliceutils.GetInt(colWidths, n); !ok || currentSize < len(header) {
			colWidths[n] = len(header) // Set width according to header size
		}
		// Check row column sizes
		for _, row := range t.rows {
			spanWidth := padding * 2
			if rowValue, ok := sliceutils.GetString(row.columns, n); ok && colWidths[n] < len(rowValue) {
				if len(row.columns) < len(t.headers) {
					spanWidth += len(rowValue) // This is a spanned column, so its width does not apply to the individual column
				} else {
					colWidths[n] = len(rowValue) // Set width according to column size
				}
			}
			if spanWidth > minWidth {
				minWidth = spanWidth
			}
		}
	}

	// Add padding and calculate the total width according to the column sizes (disregards spanned columns)
	columnTotal := 0
	for n, w := range colWidths {
		colWidths[n] = w + (padding * 2)
		if colWidths[n] > maxTotalWidth {
			colWidths[n] = maxTotalWidth
		}
		columnTotal += colWidths[n]
	}

	// Equalize widths by 20% of the average
	// This is to prevent columns that are much larger than others from taking up most of the table width
	averageWidth := 20 / len(colWidths)
	columnTotal += averageWidth * len(colWidths)
	for n := range colWidths {
		colWidths[n] += averageWidth
	}

	total := columnTotal

	// Factor in spanned columns
	if total < minWidth {
		total = minWidth
	}

	// Limit to max width
	if total > maxTotalWidth {
		total = maxTotalWidth
	}

	// Calculate column widths according to the total width
	calculatedTotal := 0
	for n, w := range colWidths {
		colWidths[n] = int(math.Floor(float64(w) / float64(columnTotal) * float64(total)))
		calculatedTotal += colWidths[n]
	}
	colWidths[len(colWidths)-1] += total - calculatedTotal // Ensure we use up all remaining space

	logging.Debug("Table column widths: %v, total: %d", colWidths, total)

	return colWidths, total
}

func renderRow(providedColumns []string, colWidths []int) string {
	// don't want to modify the provided slice
	columns := make([]string, len(providedColumns))
	copy(columns, providedColumns)

	result := ""

	// Keep rendering lines until there's no column data left to render
	for len(strings.Join(columns, "")) != 0 {
		// Iterate over the columns by their line sizes
		for n, maxlen := range colWidths {
			if len(columns) < n+1 {
				continue
			}

			colValue := columns[n]

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

			breakpos := strings.Index(colValue, linebreak)
			if breakpos != -1 && breakpos < end {
				end = breakpos + 1
			}

			suffix := strings.Repeat(" ", maxlen-end)
			result += pad(colValue[0:end] + suffix)
			columns[n] = colValue[end:]
		}
		result = strings.TrimRight(result, linebreak) + linebreak
	}

	return strings.TrimRight(result, linebreak)
}

func pad(v string) string {
	padded := strings.Repeat(" ", padding)
	return padded + v + padded
}
