package exiter

import "fmt"

var returnedExitCode int

// WaitForExit will call the supplied function and return the exit code that occurs during its invokation, or -1 if no
// exit was called. This requires you to use exiter.Exit as your exit function.
func WaitForExit(f func()) (exitCode int) {
	returnedExitCode = -1
	defer func() {
		if r := recover(); r != nil {
			if fmt.Sprintf("%v", r) != "exiter" {
				panic(r)
			}
			exitCode = returnedExitCode
		}
	}()
	f()
	return returnedExitCode
}

// Exit mocks os.Exit for use with WaitForExit
func Exit(code int) {
	returnedExitCode = code
	panic("exiter")
}
