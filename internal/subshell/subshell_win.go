//go:build windows
// +build windows

package subshell

import "github.com/ActiveState/cli/internal/subshell/cmd"

func init() {
	supportedShells = []SubShell{
		&cmd.SubShell{},
	}
}
