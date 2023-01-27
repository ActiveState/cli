//go:build linux || darwin
// +build linux darwin

package output

import (
	"fmt"
)

const moveCaretBack = "\x1b[%dD" // %d is the number of characters to move back

func (d *Spinner) moveCaretBackInTerminal(n int) {
	d.out.Fprint(d.out.Config().ErrWriter, fmt.Sprintf(moveCaretBack, n))
}
