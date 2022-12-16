//go:build !windows
// +build !windows

package subshell

import (
	"github.com/ActiveState/cli/internal-as/subshell/bash"
	"github.com/ActiveState/cli/internal-as/subshell/cmd"
	"github.com/ActiveState/cli/internal-as/subshell/fish"
	"github.com/ActiveState/cli/internal-as/subshell/tcsh"
	"github.com/ActiveState/cli/internal-as/subshell/zsh"
)

var supportedShells = []SubShell{
	&bash.SubShell{},
	&zsh.SubShell{},
	&tcsh.SubShell{},
	&fish.SubShell{},
	&cmd.SubShell{},
}
