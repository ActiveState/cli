package sscommon

import (
	"os/exec"
	"syscall"
)

var lineBreak = "\n"
var lineBreakChar = `\n`

func stop(cmd *exec.Cmd) error {
	// may panic if process no longer exists
	defer failures.Recover()

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
