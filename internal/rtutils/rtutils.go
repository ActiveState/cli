package rtutils

import (
	"fmt"
	"runtime"
	"time"
)

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

// Closer is a convenience function that addresses the use-case of wanting to defer a Close() method that returns an error
// By using this function you can pass it the error the function returned as the second argument, if both the closer
// and the function error are not-nil the function error will get wrapped by the closer error, albeit with a new error
// struct, so the types and parent structure of the closer error would be lost if you use this function
func Closer(closer func() error, rerr *error) {
	err := closer()
	if err != nil {
		if *rerr != nil {
			*rerr = fmt.Errorf("%s: %w", err.Error(), *rerr)
		} else {
			*rerr = err
		}
	}
}

var ErrTimeout = fmt.Errorf("Timed out")

func Timeout(cb func() error, t time.Duration) error {
	err := make(chan error, 1)
	go func() {
		err <- cb()
	}()
	select {
	case errv := <-err:
		return errv
	case <-time.After(t):
		return ErrTimeout
	}
}