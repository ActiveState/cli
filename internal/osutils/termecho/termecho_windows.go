package termecho

import (
	"os"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"golang.org/x/sys/windows"
)

func toggle(on bool) error {
	fd := windows.Handle(os.Stdin.Fd())
	var mode uint32
	err := windows.GetConsoleMode(fd, &mode)
	if err != nil {
		if shell := os.Getenv("SHELL"); shell != "" {
			logging.Debug("Cannot turn off terminal echo in %s", shell)
			return nil
		}
		return errs.Wrap(err, "Error calling GetConsoleMode")
	}

	newMode := mode
	if !on {
		newMode &^= windows.ENABLE_ECHO_INPUT
	} else {
		newMode |= windows.ENABLE_ECHO_INPUT
	}
	err = windows.SetConsoleMode(fd, newMode)
	if err != nil {
		return errs.Wrap(err, "Error calling SetConsoleMode")
	}

	return nil
}
