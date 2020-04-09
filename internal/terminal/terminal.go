// Package terminal includes helper functions for terminal capabilities
package terminal

import (
	"os"
	"runtime"

	"golang.org/x/crypto/ssh/terminal"
)

// fdSupportsColors implements a heuristic checking whether a file descriptor supports colors
func fdSupportsColors(fd int, lookupEnv func(string) (string, bool)) bool {
	if runtime.GOOS == "windows" {
		return true
	}
	termValue, ok := lookupEnv("TERM")
	if !ok {
		return false
	}
	if termValue == "dumb" {
		return false
	}
	return terminal.IsTerminal(fd)
}

// StdoutSupportsColors returns whether stdout supports color output
// - It always returns true on Windows
// - If the TERM variable is not set, or set to the "dumb" terminal, no color support
//   is assumed.
// - If stdout is not a terminal, color support is disabled
func StdoutSupportsColors() bool {
	return fdSupportsColors(int(os.Stdout.Fd()), os.LookupEnv)
}
