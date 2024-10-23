//go:build windows
// +build windows

package subshell

import (
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/subshell/bash"
	"github.com/ActiveState/cli/internal/subshell/cmd"
	"github.com/ActiveState/cli/internal/subshell/pwsh"
)

var supportedShells = []SubShell{
	&cmd.SubShell{},
	&pwsh.SubShell{},
	&bash.SubShell{},
}

const (
	SHELL_ENV_VAR = "COMSPEC"
	OS_DEFAULT    = "cmd.exe"
)

func supportedShellName(filename string) bool {
	for _, subshell := range supportedShells {
		if strings.EqualFold(filename, fmt.Sprintf("%s.exe", subshell.Shell())) {
			return true
		}
	}
	return false
}
