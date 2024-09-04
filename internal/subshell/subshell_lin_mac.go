//go:build !windows
// +build !windows

package subshell

import (
	"github.com/ActiveState/cli/internal/subshell/bash"
	"github.com/ActiveState/cli/internal/subshell/cmd"
	"github.com/ActiveState/cli/internal/subshell/fish"
	"github.com/ActiveState/cli/internal/subshell/tcsh"
	"github.com/ActiveState/cli/internal/subshell/zsh"
)

var supportedShells = []SubShell{
	&bash.SubShell{},
	&zsh.SubShell{},
	&tcsh.SubShell{},
	&fish.SubShell{},
	&cmd.SubShell{},
}

const (
	SHELL_ENV_VAR = "SHELL"
	OS_DEFAULT    = "bash"
)

func supportedShellName(name string) bool {
	for _, subshell := range supportedShells {
		if name == subshell.Shell() {
			return true
		}
	}
	return false
}
