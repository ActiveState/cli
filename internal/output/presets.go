package output

import (
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/colorize"
)

type Title string

func (t Title) String() string {
	return fmt.Sprintf("[DISABLED]░▒▓█[/RESET] [HEADING]%s[/RESET]", string(t))
}

func (t Title) MarshalOutput(f Format) interface{} {
	if f != PlainFormatName {
		return Suppress
	}
	return t.String()
}

type Heading string

func (h Heading) String() string {
	underline := strings.Repeat(dash, len(colorize.StripColorCodes(string(h))))
	return fmt.Sprintf("\n[HEADING]%s[/RESET]\n[DISABLED]%s[/RESET]", string(h), underline)
}

func (h Heading) MarshalOutput(f Format) interface{} {
	if f != PlainFormatName {
		return Suppress
	}
	return h.String()
}

type SubHeading string

func (h SubHeading) String() string {
	return fmt.Sprintf("\n[HEADING]%s[/RESET]", string(h))
}

func (h SubHeading) MarshalOutput(f Format) interface{} {
	if f != PlainFormatName {
		return Suppress
	}
	return h.String()
}
