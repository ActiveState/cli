package osutils

import (
	"regexp"
	"strings"
)

// ShellEscape serve to escape arguments passed to shell commands
type ShellEscape struct {
	wordPattern   *regexp.Regexp
	escapePattern *regexp.Regexp
	escapeWith    string
}

//NewBashEscaper creates a new instance of ShellEscape that's configured for escaping bash style arguments
func NewBashEscaper() *ShellEscape {
	return &ShellEscape{
		regexp.MustCompile(`^[\w]+$`),
		regexp.MustCompile(`(\\|"|\$)`),
		`\$1`,
	}
}

// NewBatchEscaper creates a new isntance of ShellEscape that's configured for escaping batch style arguments
func NewBatchEscaper() *ShellEscape {
	return &ShellEscape{
		regexp.MustCompile(`^[\w]+$`),
		regexp.MustCompile(`"`),
		`""`,
	}
}

// EscapeLineEnd will escape any line end characters that require escaping for the purpose of quoting
func (s *ShellEscape) EscapeLineEnd(value string) string {
	value = strings.Replace(value, "\n", `\n`, -1)
	value = strings.Replace(value, "\r", `\r`, -1)
	return value
}

// Escape will escape any characters that require escaping for the purpose of quoting
func (s *ShellEscape) Escape(value string) string {
	return s.escapePattern.ReplaceAllString(value, s.escapeWith)
}

// Quote implements SubShell.Quote
func (s *ShellEscape) Quote(value string) string {
	if len(value) == 0 {
		return `""`
	}
	if s.wordPattern.MatchString(value) {
		return value
	}
	return `"` + s.EscapeLineEnd(s.Escape(value)) + `"`
}
