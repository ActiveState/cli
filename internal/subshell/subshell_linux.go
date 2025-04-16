//go:build linux
// +build linux

package subshell

import (
	"strings"

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

func supportedShellName(filename string) bool {
	for _, subshell := range supportedShells {
		if strings.EqualFold(filename, subshell.Shell()) {
			return true
		}
	}
	return false
}
