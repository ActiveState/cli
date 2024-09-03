//go:build windows
// +build windows

package e2e

import (
	"runtime"
	"time"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/termtest"
)

var (
	RuntimeSourcingTimeoutOpt      = termtest.OptExpectTimeout(3 * time.Minute)
	RuntimeBuildSourcingTimeoutOpt = termtest.OptExpectTimeout(6 * time.Minute)
)

// SpawnInsideShell spawns the state tool executable to be tested with arguments.
// This function differs from Spawn in that it runs the command in a shell on Windows CI.
// Our Windows integration tests are run on bash. Due to the way the PTY library runs a new
// command we need to run the command inside a shell in order to setup the correct process
// tree. Without this integration tests run in bash will incorrectly identify the partent shell
// as bash, rather than the actual shell that is running the command
func (s *Session) SpawnInsideShell(args ...string) *SpawnedCmd {
	opts := []SpawnOptSetter{OptArgs(args...)}
	if runtime.GOOS == "windows" && condition.OnCI() {
		opts = append(opts, OptRunInsideShell(true))
	}
	return s.SpawnCmdWithOpts(s.Exe, opts...)
}

// SpawnCmdInsideShellWithOpts spawns the executable to be tested with arguments and options.
// This function differs from SpawnCmdWithOpts in that it runs the command in a shell on Windows CI.
// See SpawnInsideShell for more information.
func (s *Session) SpawnCmdInsideShellWithOpts(exe string, opts ...SpawnOptSetter) *SpawnedCmd {
	if runtime.GOOS == "windows" && condition.OnCI() {
		opts = append(opts, OptRunInsideShell(true))
	}
	return s.SpawnCmdWithOpts(exe, opts...)
}

// SpawnInsideShell spawns the state tool executable to be tested with arguments.
// This function differs from SpawnWithOpts in that it runs the command in a shell on Windows CI.
// See SpawnInsideShell for more information.
func (s *Session) SpawnInsideShellWithOpts(opts ...SpawnOptSetter) *SpawnedCmd {
	if runtime.GOOS == "windows" && condition.OnCI() {
		opts = append(opts, OptRunInsideShell(true))
	}
	return s.SpawnCmdWithOpts(s.Exe, opts...)
}
