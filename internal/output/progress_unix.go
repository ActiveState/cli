//go:build linux || darwin
// +build linux darwin

package output

import (
	"github.com/ActiveState/cli/internal/rollbar"
)

func (d *Spinner) moveCaretBackInCommandPrompt(n int) {
	if !d.reportedError {
		rollbar.Error("Incorrectly detected Windows command prompt in Unix environment")
		d.reportedError = true
	}
}
