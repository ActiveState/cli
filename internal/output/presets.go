package output

import (
	"fmt"
	"strings"
)

type Heading string

func (h Heading) MarshalOutput(f Format) interface{} {
	if f != PlainFormatName {
		return Suppress
	}
	underline := strings.Repeat(dash, len(h))
	return fmt.Sprintf("\n[HEADING]%s[/RESET]\n[DISABLED]%s[/RESET]", h, underline)
}
