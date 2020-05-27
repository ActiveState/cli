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
	colorRx, err = regexp.Compile(`\[(BOLD|UNDERLINE|BLACK|RED|GREEN|YELLOW|BLUE|MAGENTA|CYAN|WHITE|INFO|/RESET)!?\]`)
	if err != nil {
		panic(fmt.Sprintf("Could not compile regex: %v", err))
	}
}

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
			brighten := value[end-2:end-1] == "!"
			groupName := value[groupStart:groupEnd]
			colorize(writer, groupName, brighten)
		}

		pos = end
	}

	return writer.Write([]byte(value[pos:len(value)]))
}

// StripColorCodes strips color codes from a string
func StripColorCodes(value string) string {
	return colorRx.ReplaceAllString(value, "")
}

func colorize(writer io.Writer, colorName string, brighten bool) {
	switch colorName {
	case `BOLD`:
		ct.ChangeStyle(writer, ct.Bold)
	case `UNDERLINE`:
		ct.ChangeStyle(writer, ct.Underline)
	case `BLACK`:
		ct.Foreground(writer, ct.Black, brighten)
	case `RED`:
		ct.Foreground(writer, ct.Red, brighten)
	case `GREEN`:
		ct.Foreground(writer, ct.Green, brighten)
	case `YELLOW`:
		ct.Foreground(writer, ct.Yellow, brighten)
	case `BLUE`:
		ct.Foreground(writer, ct.Blue, brighten)
	case `MAGENTA`:
		ct.Foreground(writer, ct.Magenta, brighten)
	case `CYAN`:
		ct.Foreground(writer, ct.Cyan, brighten)
	case `WHITE`:
		ct.Foreground(writer, ct.White, brighten)
	case `INFO`:
		ct.Foreground(writer, ct.Blue, brighten)
		ct.ChangeStyle(writer, ct.Bold)
	case `/RESET`:
		ct.Reset(writer)
	}
}
