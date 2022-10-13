package e2e

import (
	"strings"
	"time"

	"github.com/ActiveState/termtest"
)

type SessionConsoleProcess struct {
	*termtest.ConsoleProcess
	cp *termtest.ConsoleProcess
}

// ExpectExitCode is a wrapper around termtest.ConsoleProcess.ExpectExitCode that ignores cmd.exe,
// which misbehaves when comes to exit codes.
func (scp *SessionConsoleProcess) ExpectExitCode(exitCode int, timeout ...time.Duration) (string, error) {
	if strings.HasSuffix(scp.Executable(), "cmd.exe") {
		return "", nil
	}
	return scp.cp.ExpectExitCode(exitCode, timeout...)
}

// ExpectNotExitCode is a wrapper around termtest.ConsoleProcess.ExpectNotExitCode that ignores
// cmd.exe, which misbehaves when comes to exit codes.
func (scp *SessionConsoleProcess) ExpectNotExitCode(exitCode int, timeout ...time.Duration) (string, error) {
	if strings.HasSuffix(scp.Executable(), "cmd.exe") {
		return "", nil
	}
	return scp.cp.ExpectNotExitCode(exitCode, timeout...)
}
