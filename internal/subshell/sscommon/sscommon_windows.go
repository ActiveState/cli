package sscommon

import (
	"os/exec"
	"syscall"

	"github.com/ActiveState/cli/internal/failures"
)

func stop(cmd *exec.Cmd) *failures.Failure {
	// windows should use "CTRL_CLOSE_EVENT"; SIGKILL works
	sig := syscall.SIGKILL

	// may panic if process no longer exists
	defer failures.Recover()
	if err := cmd.Process.Signal(sig); err != nil {
		return FailSignalCmd.Wrap(err)
	}

	return nil
}

// runPrepare ensures that the processes are run in a new process group.
// The effects of this flag are explained here.
// https://docs.microsoft.com/en-us/windows/win32/procthread/process-creation-flags
// Without the flag, `state run` commands can misbehave.  The simplest example
// is a script that executes `timeout 10`.  When run, press Ctrl+C or Ctrl+Break event.
// As commands are run inside of a Batch script, the BAT scripts asks the user whether
// it wants to terminate the script.  Input however, is not redirected to the batch
// script anymore, so it just hangs around...
// To be honest, it is not clear to me, why starting it in a new process group fixes
// the issue. Perhaps without creating a new process group, the parent *also* gets
// interrupted and then detaches the stdin from the batch file. But this is just a guess...
// So, the entire thing keeps being scary.
func runPrepare(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.CreationFlags = 0x00000200 // NEW_PROCESS_GROUP
}
