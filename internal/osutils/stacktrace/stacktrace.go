package stacktrace

import (
	"fmt"
	"strings"
)

// Stacktrace represents a stacktrace
type Stacktrace struct {
	Frames []Frame
}

// Frame is a single frame in a stacktrace
type Frame struct {
	// Func contains a function name.
	Func string
	// Line contains a line number.
	Line int
	// Path contains a file path.
	Path string
	// Package is the package name for this frame
	Package string
}

// FrameCap is a default cap for frames array.
// It can be changed to number of expected frames
// for purpose of performance optimisation.
var FrameCap = 20

// String returns a string representation of a stacktrace
func (t *Stacktrace) String() string {
	result := []string{}
	for _, frame := range t.Frames {
		result = append(result, fmt.Sprintf(`%s:%s:%d`, frame.Path, frame.Func, frame.Line))
	}
	return strings.Join(result, "\n")
}

// Get returns a stacktrace
func Get() *Stacktrace {
	return GetWithSkip(nil)
}

func GetWithSkip(skipFiles []string) *Stacktrace {
	return &Stacktrace{}
	/*stacktrace := &Stacktrace{}
		pc := make([]uintptr, FrameCap)
		n := runtime.Callers(1, pc)
		if n == 0 {
			return stacktrace
		}

		pc = pc[:n]
		frames := runtime.CallersFrames(pc)
		skipFiles = append(skipFiles, rtutils.CurrentFile()) // Also skip the file we're in
	LOOP:
		for {
			frame, more := frames.Next()
			pkg := strings.Split(frame.Func.Name(), ".")[0]

			for _, skipFile := range skipFiles {
				if frame.File == skipFile {
					continue LOOP
				}
			}

			stacktrace.Frames = append(stacktrace.Frames, Frame{
				Func:    frame.Func.Name(),
				Line:    frame.Line,
				Path:    frame.File,
				Package: pkg,
			})

			if !more {
				break
			}
		}

		return stacktrace*/
}
