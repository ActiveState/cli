//go:build !windows
// +build !windows

package e2e

import (
	"time"

	"github.com/ActiveState/termtest"
)

var (
	RuntimeSourcingTimeoutOpt      = termtest.OptExpectTimeout(3 * time.Minute)
	RuntimeBuildSourcingTimeoutOpt = termtest.OptExpectTimeout(6 * time.Minute)
)

// SpawnInsideShell spawns the state tool executable to be tested with arguments.
// On Unix systems, this function is equivalent to Spawn.
func (s *Session) SpawnInsideShell(args ...string) *SpawnedCmd {
	return s.SpawnCmd(s.Exe, args...)
}

// SpawnCmdInsideShellWithOpts spawns the state tool executable to be tested with arguments and options.
// On Unix systems, this function is equivalent to SpawnCmdWithOpts.
func (s *Session) SpawnCmdInsideShellWithOpts(exe string, opts ...SpawnOptSetter) *SpawnedCmd {
	return s.SpawnCmdWithOpts(exe, opts...)
}

// SpawnInsideShell spawns the state tool executable to be tested with arguments.
// On Unix systems, this function is equivalent to SpawnWithOpts.
func (s *Session) SpawnInsideShellWithOpts(opts ...SpawnOptSetter) *SpawnedCmd {
	return s.SpawnCmdWithOpts(s.Exe, opts...)
}
