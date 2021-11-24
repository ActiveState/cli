package panics

import (
	"fmt"
	"os"

	"github.com/ActiveState/cli/internal/logging"
)

// HandlePanics produces actionable output for panic events (that shouldn't happen) and returns whether a panic event has been handled
func HandlePanics(recovered interface{}, stack []byte) bool {
	if recovered != nil {
		logging.Error("Panic: %v", recovered)
		logging.Debug("Stack: %s", string(stack))

		fmt.Fprintln(os.Stderr, fmt.Sprintf(`An unexpected error occurred while running the State Tool.
Error: %v
Check the error log for more information.
Your error log is located at: %s`, recovered, logging.FilePath()))
		return true
	}
	return false
}

// LogPanics produces actionable output for panic events (that shouldn't happen) and returns whether a panic event has been handled
func LogPanics(recovered interface{}, stack []byte) bool {
	if recovered != nil {
		logging.Error("Panic: %v", recovered)
		logging.Debug("Stack: %s", string(stack))
		return true
	}
	return false
}
