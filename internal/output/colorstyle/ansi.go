// +build !windows

package colorstyle

import (
	"io"
)

type Styler struct {
	writer io.Writer
}

var ansiStyleMap = map[Style]string{
	Default:   "\x1b[0;0",
	Reversed:  "\x1b[0;7",
	Bold:      "\x1b[0;1",
	Underline: "\x1b[0;4",
	Black:     "\x1b[0;30",
	Red:       "\x1b[0;31",
	Green:     "\x1b[0;32",
	Yellow:    "\x1b[0;33",
	Blue:      "\x1b[0;34",
	Magenta:   "\x1b[0;35",
	Cyan:      "\x1b[0;36",
	White:     "\x1b[0;37",
}

func New(writer io.Writer) *Styler {
	return &Styler{writer}
}

func (w *Styler) SetStyle(s Style, bright bool) {
	resolvedStyle := ansiStyleMap[s]
	if bright {
		resolvedStyle = resolvedStyle + ";1"
	}
	w.writer.Write([]byte(resolvedStyle + "m"))
}
