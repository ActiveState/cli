package textutils

import (
	"os"

	"github.com/ActiveState/cli/internal/logging"
	"github.com/eidolon/wordwrap"
	"golang.org/x/crypto/ssh/terminal"
)

// WordWrap wraps a block of text at word boundaries tailored to the terminal size
func WordWrap(text string) string {
	termWidth, _, err := terminal.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		logging.Debug("Cannot get terminal size: %v", err)
		termWidth = 100
	}
	f := wordwrap.Wrapper(termWidth-1, false)
	return f(text) + "\n"
}
