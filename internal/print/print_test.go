package print

import (
	"testing"
)

func TestLine(t *testing.T) {
	Line("hello world")
}

// Formatted aliases to fmt.printf, also invokes Println
func TestFormatted(t *testing.T) {
	Formatted("hello %s", "world")
	Line("") // end with newline to work around https://github.com/jstemmer/go-junit-report/issues/28
}

// Error prints the given string as an error message
func TestError(t *testing.T) {
	Error("hello world")
}

// Warning prints the given string as a warning message
func TestWarning(t *testing.T) {
	Warning("hello world")
}

// Info prints the given string as an info message
func TestInfo(t *testing.T) {
	Info("hello world")
}
