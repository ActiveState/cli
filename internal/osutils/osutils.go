package osutils

import (
	"os"
	"os/exec"

	"github.com/ActiveState/cli/internal/logging"
)

// CmdExitCode returns the exit code of a command in a platform agnostic way
// taken from https://www.reddit.com/r/golang/comments/1hvvnn/any_better_way_to_do_a_crossplatform_exec_and/caytqvr/
func CmdExitCode(cmd *exec.Cmd) (code int) {
	defer func() {
		if r := recover(); r != nil {
			logging.Errorf("Could not get exit code, so returning 1 instead (this is non-fatal, but should be resolved), actual error: %v", r)
			code = 128
		}
	}()

	type Status interface {
		ExitStatus() int
	}
	return cmd.ProcessState.Sys().(Status).ExitStatus()
}

// ExecuteAndPipeStd will run the given command and pipe stdin, stdout and stderr
func ExecuteAndPipeStd(command string, arg ...string) (int, *exec.Cmd, error) {
	logging.Debug("Executing command and piping std: %s, %v", command, arg)

	cmd := exec.Command(command, arg...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	err := cmd.Run()
	if err != nil {
		logging.Error("Executing command returned error: %v", err)
	}
	return CmdExitCode(cmd), cmd, err
}
