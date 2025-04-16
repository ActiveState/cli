//go:build darwin
// +build darwin

package subshell

import (
	"strings"

	"github.com/ActiveState/cli/internal/subshell/bash"
	"github.com/ActiveState/cli/internal/subshell/fish"
	"github.com/ActiveState/cli/internal/subshell/tcsh"
	"github.com/ActiveState/cli/internal/subshell/zsh"
)

var supportedShells = []SubShell{
	&bash.SubShell{},
	&zsh.SubShell{},
	&tcsh.SubShell{},
	&fish.SubShell{},
}

const (
	SHELL_ENV_VAR = "SHELL"
	OS_DEFAULT    = "zsh"
)

func supportedShellName(filename string) bool {
	for _, subshell := range supportedShells {
		if strings.EqualFold(filename, subshell.Shell()) {
			return true
		}
	}
	return false
}
