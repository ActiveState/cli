package termutils

import (
	"os"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/ActiveState/cli/internal/logging"
)

const fallbackWidth = 100
const maxWidth = 160

func GetWidth() int {
	termWidth, _, err := terminal.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		logging.Debug("Cannot get terminal size: %v", err)
		termWidth = fallbackWidth
	}
	if termWidth == 0 {
		termWidth = fallbackWidth
	}
	if termWidth > maxWidth {
		termWidth = maxWidth
	}
	return termWidth
}
