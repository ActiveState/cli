//go:build !windows
// +build !windows

package osutils

import (
	"os/exec"
)

// CmdExitCode returns the exit code of a command
func CmdExitCode(cmd *exec.Cmd) (code int) {
	return cmd.ProcessState.ExitCode()
}