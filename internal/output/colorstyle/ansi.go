package colorstyle

import (
	"io"
)

type ANSI struct {
	writer io.Writer
}

var ansiStyleMap = map[Style]string{
	Default:   "\x1b[0",
	Reversed:  "\x1b[7",
	Bold:      "\x1b[1",
	Underline: "\x1b[4",
	Black:     "\x1b[30",
	Red:       "\x1b[31",
	Green:     "\x1b[32",
	Yellow:    "\x1b[33",
	Blue:      "\x1b[34",
	Magenta:   "\x1b[35",
	Cyan:      "\x1b[36",
	White:     "\x1b[37",
}

func NewANSI(writer io.Writer) *ANSI {
	return &ANSI{writer}
}

func (w *ANSI) SetStyle(s Style, bright bool) {
	resolvedStyle := ansiStyleMap[s]
	if bright {
		resolvedStyle = resolvedStyle + ";1"
	}
	w.writer.Write([]byte(resolvedStyle + "m"))
}
