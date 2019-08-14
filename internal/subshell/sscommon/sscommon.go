package sscommon

import (
	"os"
	"os/exec"
	"strings"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
)

var (
	// FailExecCmd represents a failure running a cmd
	FailExecCmd = failures.Type("sscommon.fail.execcmd")

	// FailSignalCmd represents a failure sending a system signal to a cmd
	FailSignalCmd = failures.Type("sscommon.fail.signalcmd")
)

// Start wires stdin/stdout/stderr into the provided command, starts it, and
// returns a channel to monitor errors on.
func Start(cmd *exec.Cmd) chan *failures.Failure {
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	cmd.Start()

	fs := make(chan *failures.Failure, 1)

	go func() {
		defer close(fs)

		if err := cmd.Wait(); err != nil {
			if eerr, ok := err.(*exec.ExitError); ok {
				code := eerr.ExitCode()
				valid := eerr.Exited()
				// code 130 is returned when a process halts
				// due to SIGTERM after receiving a SIGINT
				// code -1 is returned when a process halts
				// due to SIGTERM without any interference.
				if code == 130 || (valid && code == -1) {
					logging.Debug("exit - valid: %t, code: %d", valid, code)
					return
				}

				fs <- FailExecCmd.Wrap(eerr)
				return
			}

			fs <- FailExecCmd.Wrap(err)
			return
		}
	}()

	return fs
}

// Stop signals the provided command to terminate.
func Stop(cmd *exec.Cmd) *failures.Failure {
	return stop(cmd)
}

// RunFunc ...
type RunFunc func(env []string, name string, args ...string) (int, error)

// RunFuncByBinary ...
func RunFuncByBinary(binary string) RunFunc {
	switch strings.ToLower(binary) {
	case "bash":
		return runWithBash
	default:
		return runDirect
	}
}

func runWithBash(env []string, name string, args ...string) (int, error) {
	filePath, fail := osutils.BashifyPath(name)
	if fail != nil {
		return 1, fail.ToError()
	}

	esc := osutils.NewBashEscaper()

	quotedArgs := esc.Quote(filePath)
	for _, arg := range args {
		quotedArgs += " " + esc.Quote(arg)
	}

	return runDirect(env, "bash", "-c", quotedArgs)
}

func runDirect(env []string, name string, args ...string) (int, error) {
	logging.Debug("Running command: %s %s", name, strings.Join(args, " "))

	runCmd := exec.Command(name, args...)
	runCmd.Stdin, runCmd.Stdout, runCmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	runCmd.Env = env

	err := runCmd.Run()
	return osutils.CmdExitCode(runCmd), err
}
