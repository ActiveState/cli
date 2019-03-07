package print

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/mattn/go-colorable"
)

// Stderr will send most print calls to stderr rather than stdout during its execution.
// This is a temporary function meant to avoid scope creep, stderr should be handled properly - https://www.pivotaltracker.com/story/show/164437345
func Stderr(call func()) {
	output := color.Output
	color.Output = colorable.NewColorableStderr()
	defer func() { color.Output = output }()
	call()
}

// Line aliases to fmt.Println
func Line(a ...interface{}) (n int, err error) {
	return fmt.Println(a...)
}

// Formatted aliases to fmt.printf, also invokes Println
func Formatted(format string, a ...interface{}) (n int, err error) {
	r, err := fmt.Printf(format, a...)

	if err != nil {
		fmt.Println()
	}

	return r, err
}

// Error prints the given string as an error message
func Error(format string, a ...interface{}) {
	color.Red(format, a...)
}

// Warning prints the given string as a warning message
func Warning(format string, a ...interface{}) {
	color.Yellow(format, a...)
}

// Info prints the given string as an info message
func Info(format string, a ...interface{}) {
	c := color.New(color.Bold, color.FgBlue)
	if len(a) == 0 {
		c.Println(format)
	} else {
		c.Printf(format, a...)
		c.Println()
	}
}

// Bold prints the given string as bolded message
func Bold(format string, a ...interface{}) {
	c := color.New(color.Bold)
	if len(a) == 0 {
		c.Println(format)
	} else {
		c.Printf(format, a...)
		c.Println()
	}
}

// BoldInline prints the given string as bolded message in line
func BoldInline(format string, a ...interface{}) {
	c := color.New(color.Bold)
	c.Printf(format, a...)
}
