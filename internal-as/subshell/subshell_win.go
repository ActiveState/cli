//go:build windows
// +build windows

package subshell

import "github.com/ActiveState/cli/internal-as/subshell/cmd"

var supportedShells = []SubShell{
	&cmd.SubShell{},
}
