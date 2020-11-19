package table

import (
	"math"
	"strings"
	"unicode/utf8"

	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/sliceutils"
	"github.com/ActiveState/cli/internal/termutils"
)

const dash = "\u2500"
const linebreak = "\n"
const linebreakRune = '\n'
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

// equalizeWidths equalizes the width of given columns by a given percentage of the average columns width
func equalizeWidths(colWidths []int, total int, percentage int) ([]int, int) {
	averageWidth := total / len(colWidths)
	equalizer := percentage * averageWidth / 100
	total += equalizer * len(colWidths)
	for n := range colWidths {
		colWidths[n] += equalizer
	}
	return colWidths, total
}

func rescaleColumns(colWidths []int, total, targetTotal int) []int {
	// Calculate column widths according to the total width
	remaining := targetTotal
	for n, w := range colWidths {
		cw := int(math.Floor(float64(w) / float64(total) * float64(remaining)))
		total -= w
		remaining -= cw
		colWidths[n] = cw
	}

	colWidths[len(colWidths)-1] += remaining // Ensure we use up all remaining space
	return colWidths
}

func (t *Table) calculateWidth(maxTotalWidth int) ([]int, int) {
	// Calculate required width of each column
	minWidth := padding * 2
	colWidths := make([]int, len(t.headers))
	columnTotal := 0
	for n, header := range t.headers {
		// Check header sizes
		headerSize := utf8.RuneCountInString(header)
		if currentSize, ok := sliceutils.GetInt(colWidths, n); !ok || currentSize < headerSize {
			colWidths[n] = headerSize // Set width according to header size
		}
		// Check row column sizes
		for _, row := range t.rows {
			spanWidth := columnTotal
			rowValueSize := 0
			if rowValue, ok := sliceutils.GetString(row.columns, n); ok {
				rowValueSize = utf8.RuneCountInString(rowValue)
			}
			if colWidths[n] < rowValueSize {
				if len(row.columns) < len(t.headers) {
					spanWidth += rowValueSize + (padding * 2) // This is a spanned column, so its width does not apply to the individual column
				} else {
					colWidths[n] = rowValueSize // Set width according to column size
				}
			}
			if spanWidth > minWidth {
				minWidth = spanWidth
			}
		}

		// Add padding and update the total width so far
		colWidths[n] += padding * 2
		columnTotal += colWidths[n]
	}

	// Equalize widths by 20% of average width
	// This is to prevent columns that are much larger than others from taking up most of the table width
	colWidths, columnTotal = equalizeWidths(colWidths, columnTotal, 20)

	// compute the total number of columns that we want the table to use
	targetTotal := columnTotal
	// Factor in spanned columns
	if targetTotal < minWidth {
		targetTotal = minWidth
	}

	// Limit to max width
	if targetTotal > maxTotalWidth {
		targetTotal = maxTotalWidth
	}

	colWidths = rescaleColumns(colWidths, columnTotal, targetTotal)
	logging.Debug("Table column widths: %v, total: %d", colWidths, targetTotal)

	return colWidths, targetTotal
}

func renderRow(providedColumns []string, colWidths []int) string {
	// don't want to modify the provided slice
	columns := make([]string, len(providedColumns))
	copy(columns, providedColumns)

	result := ""

	// Keep rendering lines until there's no column data left to render
	for len(strings.Join(columns, "")) != 0 {
		// Iterate over the columns by their line sizes
		for n, maxLen := range colWidths {
			// ignore columns that we do not have data for (they have been filled up with the last colValue already)
			if len(columns) < n+1 {
				continue
			}

			colValue := []rune(columns[n])

			// Detect multi column span
			if len(colWidths) > n+1 && len(columns) == n+1 {
				for _, v := range colWidths[n+1:] {
					maxLen += v
				}
			}

			maxLen = maxLen - (padding * 2)

			// How much of the colValue are we using for this line?
			end := len(colValue)
			if end > maxLen {
				end = maxLen
			}

			if breakpos := runeSliceIndexOf(colValue, linebreakRune); breakpos != -1 && breakpos < end {
				end = breakpos + 1
			}

			suffix := strings.Repeat(" ", maxLen-end)
			result += pad(string(colValue[0:end]) + suffix)
			columns[n] = string(colValue[end:])
		}
		result = strings.TrimRight(result, linebreak) + linebreak
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
