package exiter

import (
	"fmt"

	"github.com/kami-zh/go-capturer"
)

var defaultExiter *Exiter

func init() {
	defaultExiter = New()
}

// Exiter should be used with New
type Exiter struct {
	exitCode int
}

// New creates a new Exiter() instance
func New() *Exiter {
	return &Exiter{}
}

// Exit mocks os.Exit for use with WaitForExit
func (e *Exiter) Exit(code int) {
	e.exitCode = code
	panic("exiter")
}

// WaitForExit will call the supplied function and return the exit code that occurs during its invocation, or -1 if no
// exit was called. This requires you to use exiter.Exit as your exit function.
// WARNING - this is not threadsafe!
func (e *Exiter) WaitForExit(f func()) (exitCode int) {
	e.exitCode = -1
	defer func() {
		if r := recover(); r != nil {
			if fmt.Sprintf("%v", r) != "exiter" {
				panic(r)
			}
			exitCode = e.exitCode
		}
	}()
	f()
	return e.exitCode
}

// Capture will capture the output and exit code
func (e *Exiter) Capture(f func()) (string, int) {
	var code int
	out := capturer.CaptureOutput(func() {
		code = e.WaitForExit(f)
	})
	return out, code
}

// WaitForExit runs Exiter.WaitForExit()
func WaitForExit(f func()) (exitCode int) {
	return defaultExiter.WaitForExit(f)
}

// Capture runs Exiter.Capture()
func Capture(f func()) (string, int) {
	return defaultExiter.Capture(f)
}

// Exit runs Exiter.Exit
func Exit(code int) {
	defaultExiter.Exit(code)
}
