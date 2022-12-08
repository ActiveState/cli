package panics

import (
	"fmt"
	"os"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
)

var RecoverMessage = "state_tool_panic_recovery"

// HandlePanics produces actionable output for panic events (that shouldn't happen) and returns whether a panic event has been handled
func HandlePanics(recovered interface{}, stack []byte) bool {
	if recovered != nil {
		multilog.Error("Panic: %v", recovered)
		logging.Debug("Stack: %s", string(stack))

		fmt.Fprintln(os.Stderr, locale.Tl(RecoverMessage, "", fmt.Sprintf("%v", recovered), string(stack), logging.FilePath(), constants.ForumsURL))
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
