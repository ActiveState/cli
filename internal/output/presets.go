package output

import (
	"fmt"
)

type Title string

func (t Title) String() string {
	return fmt.Sprintf("[HEADING]█ %s[/RESET]\n", string(t))
}

func (t Title) MarshalOutput(f Format) interface{} {
	if f != PlainFormatName {
		return Suppress
	}
	return t.String()
}

type Emphasize string

func (h Emphasize) String() string {
	return fmt.Sprintf("\n[HEADING]%s[/RESET]", string(h))
}

func (h Emphasize) MarshalOutput(f Format) interface{} {
	if f != PlainFormatName {
		return Suppress
	}
	return h.String()
}
