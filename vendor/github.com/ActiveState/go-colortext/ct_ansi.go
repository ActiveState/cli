// +build !windows

package ct

import (
	"fmt"
	"io"
	"os"
	"strconv"
)

func isDumbTerm() bool {
	return os.Getenv("TERM") == "dumb"
}

func reset(writer io.Writer) {
	if isDumbTerm() {
		return
	}
	fmt.Fprint(writer, "\x1b[0m")
}

func ansiText(fg Color, fgBright bool, bg Color, bgBright bool) string {
	if fg == None && bg == None {
		return ""
	}
	s := []byte("\x1b[0")
	if fg != None {
		s = strconv.AppendUint(append(s, ";"...), 30+(uint64)(fg-Black), 10)
		if fgBright {
			s = append(s, ";1"...)
		}
	}
	if bg != None {
		s = strconv.AppendUint(append(s, ";"...), 40+(uint64)(bg-Black), 10)
		if bgBright {
			s = append(s, ";1"...)
		}
	}
	s = append(s, "m"...)
	return string(s)
}

func changeStyle(writer io.Writer, styles ...Style) {
	for _, style := range styles {
		switch style {
		case Bold:
			fmt.Fprint(writer, "\x1b[1m")
			break
		case Underline:
			fmt.Fprint(writer, "\x1b[4m")
			break
		}
	}
}

func changeColor(writer io.Writer, fg Color, fgBright bool, bg Color, bgBright bool) {
	if isDumbTerm() {
		return
	}
	if fg == None && bg == None {
		return
	}
	fmt.Fprint(writer, ansiText(fg, fgBright, bg, bgBright))
}
