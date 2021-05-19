package runbits

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/ActiveState/cli/internal/logging"
)

// HandlePanics produces actionable output for panic events (that shouldn't happen) and returns whether a panic event has been handled
func HandlePanics() bool {
	if r := recover(); r != nil {
		logging.Error("Panic: %v", r)
		logging.Debug("Stack: %s", string(debug.Stack()))

		fmt.Fprintln(os.Stderr, fmt.Sprintf(`An unexpected error occurred while running the State Tool.
Check the error log for more information.
Your error log is located at: %s`, logging.FilePath()))
		return true
	}
	return false
}
