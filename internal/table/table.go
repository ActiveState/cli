package table

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/ActiveState/cli/internal/logging"
	"golang.org/x/crypto/ssh/terminal"
)

const dash = "\u2500"

type FormatFunc func(string, ...interface{}) string

type Table struct {
	rows   [][]string
	header []string
	// TODO: Make column width part of a row struct?
	columnWidths []int
	padding      int
	// TODO: Add logic to deal with max column size
	maxSize int

	headerFormatter FormatFunc
	headerFormat    string
}

type writeBuffer struct {
	bytes.Buffer
}

func createBuffer() *writeBuffer {
	return &writeBuffer{}
}

func (b *writeBuffer) Write(str string, count int) *writeBuffer {
	for i := 0; i < count; i++ {
		b.WriteString(str)
	}
	return b
}

func (b *writeBuffer) String() string {
	return b.String()
}

func Create(data interface{}) *Table {
	t := Table{
		// Default padding value
		padding:      4,
		headerFormat: "%s",
	}

	switch v := data.(type) {
	case [][]string:
		data := data.([][]string)
		rows := make([][]string, len(data))
		for i, element := range data {
			rows[i] = element
		}
		t.rows = rows
	default:
		// TODO: Do something else
		fmt.Println(v)
	}

	return &t
}

func (t *Table) AddHeader(header []string) {
	t.header = header
}

func (t *Table) SetPadding(padding int) {
	t.padding = padding
}

func (t *Table) WithHeaderFormatter(format FormatFunc) {
	t.headerFormatter = format
}

func (t *Table) WithHeaderFormat(format string) {
	t.headerFormat = format
}

func (t *Table) Render() (string, error) {
	if len(t.rows) == 0 {
		return "", nil
	}

	// TODO: Deal with max terminal width
	// termWidth := t.getWidth()

	t.calculateWidths()
	out := t.renderHeader()
	out += t.renderRows()
	return out, nil
}

func (t *Table) calculateWidths() {
	t.columnWidths = make([]int, len(t.header))
	for i, v := range t.header {
		if w := utf8.RuneCountInString(v) + t.padding; w > t.columnWidths[i] {
			t.columnWidths[i] = w
		}
	}

	for _, row := range t.rows {
		for i, v := range row {
			if w := utf8.RuneCountInString(v) + t.padding; w > t.columnWidths[i] {
				t.columnWidths[i] = w
			}
		}
	}
}

func (t *Table) renderHeader() string {
	var out string
	for i, header := range t.header {
		formatted := fmt.Sprintf(t.headerFormat, header)
		out += formatted + strings.Repeat(" ", t.columnWidths[i]-utf8.RuneCountInString(header))
		out += t.padRow(out)
	}

	out += "\n"
	for i := range t.columnWidths {
		out += strings.Repeat(dash, t.columnWidths[i])
	}
	out += "\n"

	return out
}

func (t *Table) renderRows() string {
	var out string
	for _, row := range t.rows {
		for i, data := range row {
			out += data + strings.Repeat(" ", t.columnWidths[i]-utf8.RuneCountInString(data))
			out += t.padRow(out)
		}
		out += "\n"
	}
	return out
}

func (t *Table) padRow(row string) string {
	// TODO: Use different value for padding and spacing b/w rows
	b := createBuffer()
	b.Write(" ", t.padding)
	b.Write(row, 1)
	b.Write(" ", t.padding)
	return b.String()
}

func (t *Table) getWidth() int {
	termWidth, _, err := terminal.GetSize(int(os.Stdout.Fd()))
	if err != nil || termWidth == 0 {
		// TODO: return err?
		logging.Debug("Cannot get terminal size: %v", err)
		termWidth = 100
	}
	termWidth = termWidth - (len(t.header) * 10) // Account for cell padding, cause tabulate doesn't..
	return termWidth
}
