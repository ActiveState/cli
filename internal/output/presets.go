package output

import (
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/output/txtstyle"
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

type Title string

func (t Title) String() string {
	return txtstyle.NewTitle(string(t)).String()
}

func (t Title) MarshalOutput(f Format) interface{} {
	return txtstyle.NewTitle(string(t)).MarshalOutput(f)
}
