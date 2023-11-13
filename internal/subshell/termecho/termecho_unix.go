//go:build linux || darwin
// +build linux darwin

package termecho

import (
	"os"

	"github.com/ActiveState/cli/internal/errs"
	"golang.org/x/sys/unix"
)

func toggle(on bool) error {
	fd := int(os.Stdin.Fd())
	termios, err := unix.IoctlGetTermios(fd, ioctlReadTermios)
	if err != nil {
		return errs.Wrap(err, "Could not get termios")
	}

	newState := *termios // copy
	if !on {
		newState.Lflag &^= unix.ECHO
	} else {
		newState.Lflag |= unix.ECHO
	}
	err = unix.IoctlSetTermios(fd, ioctlWriteTermios, &newState)
	if err != nil {
		return errs.Wrap(err, "Could not set termios")
	}

	return nil
}
