package main

import (
	"os/exec"
	"syscall"
)

func Command(name string, arg ...string) *exec.Cmd {
	cmd := exec.Command(name, arg...)

	// Go currently does not escape arguments properly on Windows, it account for spaces and tab characters, but not
	// other characters that need escaping such as `<` and `>`.
	// This can be dropped once we update to a Go version that fixes this bug: https://github.com/golang/go/issues/68313
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: makeCmdLine(cmd.Args)}

	return cmd
}

// makeCmdLine builds a command line out of args by escaping "special"
// characters and joining the arguments with spaces.
// Based on syscall\exec_windows.go
func makeCmdLine(args []string) string {
	var b []byte
	for _, v := range args {
		if len(b) > 0 {
			b = append(b, ' ')
		}
		b = appendEscapeArg(b, v)
	}
	return string(b)
}

// appendEscapeArg escapes the string s, as per escapeArg,
// appends the result to b, and returns the updated slice.
// Based on syscall\exec_windows.go
func appendEscapeArg(b []byte, s string) []byte {
	if len(s) == 0 {
		return append(b, `""`...)
	}

	needsBackslash := false
	needsQuotes := false
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '"', '\\':
			needsBackslash = true
		// Based on https://github.com/sebres/PoC/blob/master/SB-0D-001-win-exec/SOLUTION.md#definition
		case ' ', '\t', '<', '>', '&', '|', '^', '!', '(', ')', '%':
			needsQuotes = true
		}
	}

	if !needsBackslash && !needsQuotes {
		// No special handling required; normal case.
		return append(b, s...)
	}
	if !needsBackslash {
		// hasSpace is true, so we need to quote the string.
		b = append(b, '"')
		b = append(b, s...)
		return append(b, '"')
	}

	if needsQuotes {
		b = append(b, '"')
	}
	slashes := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		default:
			slashes = 0
		case '\\':
			slashes++
		case '"':
			for ; slashes > 0; slashes-- {
				b = append(b, '\\')
			}
			b = append(b, '\\')
		}
		b = append(b, c)
	}
	if needsQuotes {
		for ; slashes > 0; slashes-- {
			b = append(b, '\\')
		}
		b = append(b, '"')
	}

	return b
}
