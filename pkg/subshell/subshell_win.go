//go:build windows
// +build windows

package subshell

import (
	"github.com/ActiveState/cli/pkg/subshell/cmd"
)

var supportedShells = []SubShell{
	&cmd.SubShell{},
}
