package output

import (
	"io"
	"regexp"

	"github.com/ActiveState/cli/internal/logging"
	ct "github.com/ActiveState/go-colortext"
)

func writeColorized(value string, writer io.Writer) {
	r, err := regexp.Compile(`\[(BOLD|UNDERLINE|BLACK|RED|GREEN|YELLOW|BLUE|MAGENTA|CYAN|WHITE|/RESET)!?\]`)
	if err != nil {
		logging.Errorf("Could not compile regex: %v", err)
		writer.Write([]byte(value)) // write as is
	}

	pos := 0
	matches := r.FindAllStringSubmatchIndex(value, -1)
	for _, match := range matches {
		start, end, groupStart, groupEnd := match[0], match[1], match[2], match[3]
		writer.Write([]byte(value[pos:start]))

		bright := value[end-2:end-1] == "!"
		groupName := value[groupStart:groupEnd]
		switch groupName {
		case `BOLD`:
			ct.ChangeStyle(writer, ct.Bold)
		case `UNDERLINE`:
			ct.ChangeStyle(writer, ct.Underline)
		case `BLACK`:
			ct.Foreground(writer, ct.Black, bright)
		case `RED`:
			ct.Foreground(writer, ct.Red, bright)
		case `GREEN`:
			ct.Foreground(writer, ct.Green, bright)
		case `YELLOW`:
			ct.Foreground(writer, ct.Yellow, bright)
		case `BLUE`:
			ct.Foreground(writer, ct.Blue, bright)
		case `MAGENTA`:
			ct.Foreground(writer, ct.Magenta, bright)
		case `CYAN`:
			ct.Foreground(writer, ct.Cyan, bright)
		case `WHITE`:
			ct.Foreground(writer, ct.White, bright)
		case `/RESET`:
			ct.Reset(writer)
		}

		pos = end
	}

	writer.Write([]byte(value[pos:len(value)]))
}
