//go:build !windows
// +build !windows

package subshell

import (
	"github.com/ActiveState/cli/pkg/subshell/bash"
	"github.com/ActiveState/cli/pkg/subshell/cmd"
	"github.com/ActiveState/cli/pkg/subshell/fish"
	"github.com/ActiveState/cli/pkg/subshell/tcsh"
	"github.com/ActiveState/cli/pkg/subshell/zsh"
)

var supportedShells = []SubShell{
	&bash.SubShell{},
	&zsh.SubShell{},
	&tcsh.SubShell{},
	&fish.SubShell{},
	&cmd.SubShell{},
}
