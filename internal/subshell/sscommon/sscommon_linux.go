package sscommon

import (
	"os/exec"
	"syscall"

	"github.com/ActiveState/cli/internal/errs"
)

var lineBreak = "\n"
var lineBreakChar = `\n`

func stop(cmd *exec.Cmd) error {
	sig := syscall.SIGHUP
	if err := cmd.Process.Signal(sig); err != nil {
		return errs.Wrap(err, "SignalCmd failure")
	}

	sig = syscall.SIGTERM
	if err := cmd.Process.Signal(sig); err != nil {
		return errs.Wrap(err, "SignalCmd failure")
	}

	return nil
}
