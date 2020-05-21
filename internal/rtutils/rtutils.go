package rtutils

import "runtime"

// Returns path of currently running Go file
func CurrentFile() string {
	pc := make([]uintptr, 2)
	n := runtime.Callers(1, pc)
	if n == 0 {
		return ""
	}

	pc = pc[:n]
	frames := runtime.CallersFrames(pc)

	frame, _ := frames.Next()
	frame, _ = frames.Next() // Skip rtutils.go

	return frame.File
}
