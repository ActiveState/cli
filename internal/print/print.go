package print

import (
	"fmt"

	"github.com/fatih/color"
)

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
	Bold("==> "+format, a...)
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
