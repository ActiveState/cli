package panics

import (
	"fmt"
	"os"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/logging"
)

// HandlePanics produces actionable output for panic events (that shouldn't happen) and returns whether a panic event has been handled
func HandlePanics(recovered interface{}, stack []byte) bool {
	if recovered != nil {
		logging.Error("Panic: %v", recovered)
		logging.Debug("Stack: %s", string(stack))

		fmt.Fprintln(os.Stderr, fmt.Sprintf(`An unexpected error occurred while running the State Tool.
Error: %v
Stack trace: %s
Check the error log for more information: %s
Please consider reportins your issue on the forums: %s`, recovered, string(stack), logging.FilePath(), constants.ForumsURL))
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
