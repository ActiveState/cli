package table

import (
	"strings"
	"unicode/utf8"

	"github.com/ActiveState/cli/internal/colorize"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/mathutils"
	"github.com/ActiveState/cli/internal/sliceutils"
	"github.com/ActiveState/cli/internal/termutils"
)

const dash = "\u2500"
const linebreak = "\n"
const linebreakRune = '\n'
const padding = 2

type FormatFunc func(string, ...interface{}) string

type entry struct {
	line   string
	length int
}

type row struct {
	columns []string
}

type Table struct {
	headers []string
	rows    []row

	HideHeaders bool
}

func New(headers []string) *Table {
	return &Table{headers: headers}
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

	var out string
	if !t.HideHeaders {
		out += "[NOTICE]" + renderRow(t.headers, colWidths) + "[/RESET]" + linebreak
		out += "[DISABLED]" + strings.Repeat(dash, total) + "[/RESET]" + linebreak
	}
	for _, row := range t.rows {
		out += renderRow(row.columns, colWidths) + linebreak
	}

	return strings.TrimRight(out, linebreak)
}

func (t *Table) calculateWidth(maxTableWidth int) ([]int, int) {
	// Calculate total width of each column, not worrying about max width just yet
	minTableWidth := padding * 2
	colWidths := make([]int, len(t.headers))
	colWidthsCombined := 0
	for n, header := range t.headers {
		// Start with the header size
		colWidths[n] = utf8.RuneCountInString(header)

		// Check column sizes for each row
		for _, row := range t.rows {
			columnValue, ok := sliceutils.GetString(row.columns, n)
			if !ok {
				continue // column doesn't exit because the previous column spans
			}
			// Strip any colour tags so they are not included in the width calculation
			columnValue = colorize.StripColorCodes(columnValue)
			columnSize := utf8.RuneCountInString(columnValue)

			// Detect spanned column info
			rowHasSpannedColumn := len(row.columns) < len(t.headers)
			spannedColumnIndex := len(row.columns) - 1

			if rowHasSpannedColumn && n == spannedColumnIndex {
				// Record total row size as minTableWidth
				colWidthBefore := mathutils.Total(sliceutils.IntRangeUncapped(colWidths, 0, n)...)
				minTableWidth = mathutils.MaxInt(minTableWidth, colWidthBefore+columnSize+(padding*2))
			} else {
				// This is a regular non-spanned column
				colWidths[n] = mathutils.MaxInt(colWidths[n], columnSize)
			}
		}

		// Add padding and update the total width so far
		colWidths[n] += padding * 2
		colWidthsCombined += colWidths[n]
	}

	if colWidthsCombined >= maxTableWidth {
		// Equalize widths by 20% of average width.
		// This is to prevent columns that are much larger than others
		// from taking up most of the table width.
		equalizeWidths(colWidths, 20)
	}

	// Constrain table to max and min dimensions
	tableWidth := mathutils.MaxInt(colWidthsCombined, minTableWidth)
	tableWidth = mathutils.MinInt(tableWidth, maxTableWidth)

	// Now scale back the row sizes according to the max width
	rescaleColumns(colWidths, tableWidth)
	logging.Debug("Table column widths: %v, total: %d", colWidths, tableWidth)

	return colWidths, tableWidth
}

// equalizeWidths equalizes the width of given columns by a given percentage of the average columns width
func equalizeWidths(colWidths []int, percentage int) {
	total := float64(mathutils.Total(colWidths...))
	multiplier := float64(percentage) / 100
	averageWidth := total / float64(len(colWidths))

	for n := range colWidths {
		colWidth := float64(colWidths[n])
		colWidths[n] += int((averageWidth - colWidth) * multiplier)
	}

	// Account for floats that got rounded
	if len(colWidths) > 0 {
		colWidths[len(colWidths)-1] += int(total) - mathutils.Total(colWidths...)
	}
}

func rescaleColumns(colWidths []int, targetTotal int) {
	total := float64(mathutils.Total(colWidths...))
	multiplier := float64(targetTotal) / total

	for n := range colWidths {
		colWidths[n] = int(float64(colWidths[n]) * multiplier)
	}

	// Account for floats that got rounded
	if len(colWidths) > 0 {
		colWidths[len(colWidths)-1] += targetTotal - mathutils.Total(colWidths...)
	}
}

func renderRow(providedColumns []string, colWidths []int) string {
	// Do not modify the original column widths
	widths := make([]int, len(providedColumns))
	copy(widths, colWidths)

	// Combine column widths if we have a spanned column
	if len(widths) < len(colWidths) {
		widths[len(widths)-1] = mathutils.Total(colWidths[len(widths)-1 : len(colWidths)]...)
	}

	croppedColumns := []colorize.CroppedLines{}
	for n, column := range providedColumns {
		croppedColumns = append(croppedColumns, colorize.GetCroppedText(column, widths[n]-(padding*2), false))
	}

	var rendered = true
	var lines []string
	// Iterate over rows until we reach a row where no column has data
	for lineNo := 0; rendered; lineNo++ {
		rendered = false
		var line string
		for columnNo, column := range croppedColumns {
			if lineNo > len(column)-1 {
				line += strings.Repeat(" ", widths[columnNo]) // empty column
				continue
			}
			columnLine := column[lineNo]

			// Add padding and fill up missing whitespace
			prefix := strings.Repeat(" ", padding)
			suffix := strings.Repeat(" ", padding+(widths[columnNo]-columnLine.Length-(padding*2)))

			line += prefix + columnLine.Line + suffix
			rendered = true
		}
		if rendered {
			lines = append(lines, line)
		}
	}

	return strings.TrimRight(strings.Join(lines, linebreak), linebreak)
}

func pad(v string) string {
	padded := strings.Repeat(" ", padding)
	return padded + v + padded
}
