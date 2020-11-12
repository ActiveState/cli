package termutils

import (
	"os"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/ActiveState/cli/internal/logging"
)

func GetWidth() int {
	termWidth, _, err := terminal.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		logging.Debug("Cannot get terminal size: %v", err)
		termWidth = 100
	}
	return termWidth
}
