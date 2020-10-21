package output

import (
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/colorize"
)

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
