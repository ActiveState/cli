package panics

import (
	"fmt"
	"os"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
)

// HandlePanics produces actionable output for panic events (that shouldn't happen) and returns whether a panic event has been handled
func HandlePanics(recovered interface{}, stack []byte) bool {
	if recovered != nil {
		multilog.Error("Panic: %v", recovered)
		logging.Debug("Stack: %s", string(stack))

		fmt.Fprintf(os.Stderr, `An unexpected error occurred.
Error: %v
Stack trace: %s
Check the error log for more information: %s
Please consider reporting your issue on the forums: %s`, recovered, string(stack), logging.FilePath(), constants.ForumsURL)
		return true
	}
	return false
}

// LogPanics produces actionable output for panic events (that shouldn't happen) and returns whether a panic event has been handled
func LogPanics(recovered interface{}, stack []byte) bool {
	if recovered != nil {
		multilog.Error("Panic: %v", recovered)
		logging.Debug("Stack: %s", string(stack))
		return true
	}
	return false
}

// LogAndPanic produces actionable output for panic events (that shouldn't happen) and panics
func LogAndPanic(recovered interface{}, stack []byte) {
	if recovered != nil {
		multilog.Error("Panic: %v", recovered)
		logging.Debug("Stack: %s", string(stack))
		panic(recovered) // We're only logging the panic, not interrupting it
	}
}
