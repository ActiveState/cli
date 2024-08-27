//go:build windows
// +build windows

package subshell

import "github.com/ActiveState/cli/internal/subshell/cmd"

var supportedShells = []SubShell{
	&cmd.SubShell{},
}

const (
	SHELL_ENV_VAR = "COMSPEC"
	OS_DEFULAT    = "cmd.exe"
)
