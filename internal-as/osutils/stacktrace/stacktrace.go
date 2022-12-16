package stacktrace

import (
	"fmt"
	"go/build"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/rtutils"
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

var environmentRootPath string

func init() {
	// Note: ignore any error. It cannot be logged due to logging's dependence on this package.
	environmentRootPath, _ = environment.GetRootPath()
}

// String returns a string representation of a stacktrace
// For example:
//   ./package/file.go:123:file.func
//   ./another/package.go:456:package.(*Struct).method
//   <go>/src/runtime.s:789:runtime.func
func (t *Stacktrace) String() string {
	result := []string{}
	for _, frame := range t.Frames {
		// Shorten path from its absolute path.
		path := frame.Path
		if strings.HasPrefix(path, build.Default.GOROOT) {
			// Convert "/path/to/go/distribution/file" to "<go>/file".
			path = strings.Replace(path, build.Default.GOROOT, "<go>", 1)
		} else if environmentRootPath != "" {
			// Convert "/path/to/cli/file" to "./file".
			if relPath, err := filepath.Rel(environmentRootPath, path); err == nil {
				path = "./" + relPath
			}
		}

		// Shorten fully qualified function name to its local package name.
		funcName := frame.Func
		if index := strings.LastIndex(frame.Func, "/"); index > 0 {
			// Convert "example.com/project/package/name.func" to "name.func".
			funcName = frame.Func[index+1:]
		}

		result = append(result, fmt.Sprintf(`%s:%d:%s`, path, frame.Line, funcName))
	}
	return strings.Join(result, "\n")
}

// Get returns a stacktrace
func Get() *Stacktrace {
	return GetWithSkip(nil)
}

func GetWithSkip(skipFiles []string) *Stacktrace {
	stacktrace := &Stacktrace{}
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

	return stacktrace
}
