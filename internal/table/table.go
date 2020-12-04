package table

import (
	"fmt"
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

	// Equalize widths by 20% of average width
	// This is to prevent columns that are much larger than others from taking up most of the table width
	equalizeWidths(colWidths, 20)

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
	colWidths[len(colWidths)-1] += int(total) - mathutils.Total(colWidths...)
}

func rescaleColumns(colWidths []int, targetTotal int) {
	total := float64(mathutils.Total(colWidths...))
	multiplier := float64(targetTotal) / total

	for n := range colWidths {
		colWidths[n] = int(float64(colWidths[n]) * multiplier)
	}

	// Account for floats that got rounded
	colWidths[len(colWidths)-1] += targetTotal - mathutils.Total(colWidths...)
}

func renderRow(providedColumns []string, colWidths []int) string {
	// don't want to modify the provided slice
	columns := make([]string, len(providedColumns))
	copy(columns, providedColumns)

	result := ""

	entries := make([][]colorize.Entry, len(columns))

	widths := make([]int, len(colWidths))
	copy(widths, colWidths)
	for n, maxLen := range colWidths {
		// ignore columns that we do not have data for (they have been filled up with the last colValue already)
		if len(columns) < n+1 {
			continue
		}

		// Detect multi column span
		if len(colWidths) > n+1 && len(columns) == n+1 {
			for _, v := range colWidths[n+1:] {
				maxLen += v
			}
		}

		maxLen = maxLen - (padding * 2)
		widths[n] = maxLen

		entries[n] = colorize.GetCroppedText(columns[n], maxLen)
	}

	totalRows := 0
	for _, columnEntries := range entries {
		fmt.Println("ColumnEntries:", columnEntries)
		if len(columnEntries) > totalRows {
			totalRows = len(columnEntries)
		}
	}

	// Render each row
	for i := 0; i < totalRows; i++ {
		for n, columnEntry := range entries {
			maxLen := widths[n]
			text := ""
			suffix := strings.Repeat(" ", maxLen)

			if len(columnEntry) > i {
				repeat := maxLen - columnEntry[i].Length
				if repeat < 0 {
					repeat = padding
				}
				suffix = strings.Repeat(" ", repeat)
				text = columnEntry[i].Line
			}

			result += pad(text + suffix)
		}
		result += linebreak
	}

	return strings.TrimRight(result, linebreak)
}

func pad(v string) string {
	padded := strings.Repeat(" ", padding)
	return padded + v + padded
}

func runeSliceIndexOf(slice []rune, r rune) int {
	for i, c := range slice {
		if c == r {
			return i
		}
	}
	return -1
}
