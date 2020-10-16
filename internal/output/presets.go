package output

import (
	"fmt"
	"strings"
)

type Heading string

func (h Heading) String() string {
	underline := strings.Repeat(dash, len(h))
	return fmt.Sprintf("\n[HEADING]%s[/RESET]\n[DISABLED]%s[/RESET]", string(h), underline)
}

func (h Heading) MarshalOutput(f Format) interface{} {
	if f != PlainFormatName {
		return Suppress
	}
	return h.String()
}
