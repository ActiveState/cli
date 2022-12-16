package sscommon

import (
	"os/exec"
	"syscall"

	"github.com/ActiveState/cli/internal-as/errs"
)

var lineBreak = "\n"
var lineBreakChar = `\n`

func stop(cmd *exec.Cmd) error {
	// darwin randomly returns an error when using cmd.Process.Signal
	sig := syscall.SIGHUP
	if err := syscall.Kill(cmd.Process.Pid, sig); err != nil {
		return errs.Wrap(err, "SignalCmd failure")
	}

	sig = syscall.SIGTERM
	if err := syscall.Kill(cmd.Process.Pid, sig); err != nil {
		return errs.Wrap(err, "SignalCmd failure")
	}

	return nil
}
