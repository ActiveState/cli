package output

import (
	"fmt"
	"io"
	"regexp"

	ct "github.com/ActiveState/go-colortext"
)

var colorRx *regexp.Regexp

func init() {
	var err error
	colorRx, err = regexp.Compile(`\[(HEADING|NOTICE|INFO|ERROR|DISABLED|HIGHLIGHT|/RESET)!?\]`)
	if err != nil {
		panic(fmt.Sprintf("Could not compile regex: %v", err))
	}
}

type ColorTheme interface {
	Heading(writer io.Writer)
	Notice(writer io.Writer)
	Info(writer io.Writer)
	Error(writer io.Writer)
	Disabled(writer io.Writer)
	Highlight(writer io.Writer)
	Reset(writer io.Writer)
}

type defaultColorTheme struct{}

// Heading switches to bold and bright foreground
func (dct defaultColorTheme) Heading(writer io.Writer) {
	ct.Foreground(writer, ct.White, true)
	ct.ChangeStyle(writer, ct.Bold)
}

// Notice switches to bright foreground
func (dct defaultColorTheme) Notice(writer io.Writer) {
	ct.Foreground(writer, ct.White, true)
}

// Info switches to green foreground
func (dct defaultColorTheme) Info(writer io.Writer) {
	ct.Foreground(writer, ct.Green, false)
}

// Error switches to red foreground
func (dct defaultColorTheme) Error(writer io.Writer) {
	ct.Foreground(writer, ct.Red, false)
}

// Disabled switches to bright black foreground
func (dct defaultColorTheme) Disabled(writer io.Writer) {
	ct.Foreground(writer, ct.Black, true)
}

// Highlight switches to teal foreground
func (dct defaultColorTheme) Highlight(writer io.Writer) {
	ct.Foreground(writer, ct.Cyan, true)
}

// Highlight switches to teal foreground
func (dct defaultColorTheme) Reset(writer io.Writer) {
	ct.Reset(writer)
}

var activeColorTheme ColorTheme = defaultColorTheme{}

// writeColorized will replace `[COLORNAME]foo[/RESET]` with shell colors, or strip color tags if stripColors=true
func writeColorized(value string, writer io.Writer, stripColors bool) (int, error) {
	pos := 0
	matches := colorRx.FindAllStringSubmatchIndex(value, -1)
	for _, match := range matches {
		start, end, groupStart, groupEnd := match[0], match[1], match[2], match[3]
		n, err := writer.Write([]byte(value[pos:start]))
		if err != nil {
			return n, err
		}

		if !stripColors {
			groupName := value[groupStart:groupEnd]
			colorize(activeColorTheme, writer, groupName)
		}

		pos = end
	}

	return writer.Write([]byte(value[pos:]))
}

// StripColorCodes strips color codes from a string
func StripColorCodes(value string) string {
	return colorRx.ReplaceAllString(value, "")
}

func colorize(ct ColorTheme, writer io.Writer, arg string) {
	switch arg {
	case `HEADING`:
		ct.Heading(writer)
	case `NOTICE`:
		ct.Notice(writer)
	case `INFO`:
		ct.Info(writer)
	case `ERROR`:
		ct.Error(writer)
	case `DISABLED`:
		ct.Disabled(writer)
	case `HIGHLIGHT`:
		ct.Highlight(writer)
	case `/RESET`:
		ct.Reset(writer)
	}
}
