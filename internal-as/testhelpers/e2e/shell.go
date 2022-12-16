package e2e

import (
	"strings"

	"github.com/ActiveState/cli/internal/osutils"
)

type Shell string

const (
	Bash Shell = "bash"
	Zsh        = "zsh"
	Tcsh       = "tcsh"
	Fish       = "fish"
	Cmd        = "cmd.exe"
)

// QuoteCommand constructs and returns a command line string from the given list of arguments.
// The returned string can be passed to the given shell for evaluation.
func QuoteCommand(shell Shell, args ...string) string {
	escaper := osutils.NewBashEscaper()
	if shell == Cmd {
		escaper = osutils.NewCmdEscaper()
	}
	quotedArgs := make([]string, len(args))
	for i, arg := range args {
		quotedArgs[i] = escaper.Quote(arg)
	}
	return strings.Join(quotedArgs, " ")
}
