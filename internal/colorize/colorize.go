package colorize

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"time"

	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output/colorstyle"
	"github.com/ActiveState/cli/internal/profile"
)

var colorRx *regexp.Regexp

func init() {
	defer profile.Measure("colorize:init", time.Now())
	var err error
	colorRx, err = regexp.Compile(
		`\[(HEADING|BOLD|NOTICE|SUCCESS|ERROR|WARNING|DISABLED|ACTIONABLE|CYAN|GREEN|RED|ORANGE|YELLOW|MAGENTA|/RESET)!?\]`,
	)
	if err != nil {
		panic(fmt.Sprintf("Could not compile regex: %v", err))
	}
}

type ColorTheme interface {
	Heading(writer io.Writer)
	Bold(writer io.Writer)
	Notice(writer io.Writer)
	Success(writer io.Writer)
	Error(writer io.Writer)
	Warning(writer io.Writer)
	Disabled(writer io.Writer)
	Actionable(writer io.Writer)
	Cyan(writer io.Writer)
	Green(writer io.Writer)
	Red(writer io.Writer)
	Orange(writer io.Writer)
	Yellow(writer io.Writer)
	Magenta(writer io.Writer)
	Reset(writer io.Writer)
}

type defaultColorTheme struct{}

// Heading switches to bold and bright foreground
func (dct defaultColorTheme) Heading(writer io.Writer) {
	c := colorstyle.New(writer)
	c.SetStyle(colorstyle.Default, true)
	c.SetStyle(colorstyle.Bold, false)
}

func (dct defaultColorTheme) Bold(writer io.Writer) {
	c := colorstyle.New(writer)
	c.SetStyle(colorstyle.Bold, false)
}

// Notice switches to bright foreground
func (dct defaultColorTheme) Notice(writer io.Writer) {
	colorstyle.New(writer).SetStyle(colorstyle.Default, true)
}

// Success switches to green foreground
func (dct defaultColorTheme) Success(writer io.Writer) {
	colorstyle.New(writer).SetStyle(colorstyle.Green, false)
}

// Error switches to red foreground
func (dct defaultColorTheme) Error(writer io.Writer) {
	colorstyle.New(writer).SetStyle(colorstyle.Red, false)
}

// Warning switches to yellow foreground
func (dct defaultColorTheme) Warning(writer io.Writer) {
	colorstyle.New(writer).SetStyle(colorstyle.Yellow, true)
}

// Disabled switches to bright black foreground
func (dct defaultColorTheme) Disabled(writer io.Writer) {
	colorstyle.New(writer).SetStyle(colorstyle.Dim, false)
}

// Actionable switches to teal foreground
func (dct defaultColorTheme) Actionable(writer io.Writer) {
	colorstyle.New(writer).SetStyle(colorstyle.Cyan, true)
}

// Blue switches to blue foreground
func (dct defaultColorTheme) Cyan(writer io.Writer) {
	colorstyle.New(writer).SetStyle(colorstyle.Cyan, false)
}

// Green switches to green foreground
func (dct defaultColorTheme) Green(writer io.Writer) {
	colorstyle.New(writer).SetStyle(colorstyle.Green, false)
}

// Red switches to red foreground
func (dct defaultColorTheme) Red(writer io.Writer) {
	colorstyle.New(writer).SetStyle(colorstyle.Red, false)
}

// Orange switches to orange foreground
func (dct defaultColorTheme) Orange(writer io.Writer) {
	colorstyle.New(writer).SetStyle(colorstyle.Orange, false)
}

// Yellow switches to yellow foreground
func (dct defaultColorTheme) Yellow(writer io.Writer) {
	colorstyle.New(writer).SetStyle(colorstyle.Yellow, false)
}

// Magenta switches to magenta foreground
func (dct defaultColorTheme) Magenta(writer io.Writer) {
	colorstyle.New(writer).SetStyle(colorstyle.Magenta, false)
}

// Reset re-sets all color settings
func (dct defaultColorTheme) Reset(writer io.Writer) {
	colorstyle.New(writer).SetStyle(colorstyle.Reset, false)
}

var activeColorTheme ColorTheme = defaultColorTheme{}

// Colorize will replace `[COLORNAME]foo`[/RESET] with shell colors, or strip color tags if stripColors=true
func Colorize(value string, writer io.Writer, stripColors bool) (int, error) {
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

func Colorized(value string, stripColors bool) (string, error) {
	var out bytes.Buffer
	_, err := Colorize(value, &out, stripColors)
	return out.String(), err
}

func ColorizedOrStrip(value string, stripColors bool) string {
	var out bytes.Buffer
	_, err := Colorize(value, &out, stripColors)
	if err != nil {
		multilog.Error("Could not colorize: %s", err.Error())
		return StripColorCodes(value)
	}
	return out.String()
}

// StripColorCodes strips color codes from a string
func StripColorCodes(value string) string {
	return colorRx.ReplaceAllString(value, "")
}

func colorize(ct ColorTheme, writer io.Writer, arg string) {
	// writer.Write([]byte("[" + arg + "]")) // Uncomment to debug color tags
	switch arg {
	case `HEADING`:
		ct.Heading(writer)
	case `NOTICE`:
		ct.Notice(writer)
	case `SUCCESS`:
		ct.Success(writer)
	case `WARNING`:
		ct.Warning(writer)
	case `ERROR`:
		ct.Error(writer)
	case `BOLD`:
		ct.Bold(writer)
	case `DISABLED`:
		ct.Disabled(writer)
	case `ACTIONABLE`:
		ct.Actionable(writer)
	case `CYAN`:
		ct.Cyan(writer)
	case `GREEN`:
		ct.Green(writer)
	case `RED`:
		ct.Red(writer)
	case `ORANGE`:
		ct.Orange(writer)
	case `YELLOW`:
		ct.Yellow(writer)
	case `MAGENTA`:
		ct.Magenta(writer)
	case `/RESET`:
		ct.Reset(writer)
	}
}
