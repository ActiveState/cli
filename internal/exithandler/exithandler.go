package exithandler

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
)

type CustomHandler func(error)

// Handle takes care of handling the final (!) error and is only intended to be used from the `main()` function
// It takes care of logging the error as well as handling panics
// It takes a customHandler which is meant to take care of UX as well as for producing things like a custom exit code
// and closing open handles
func Handle(err error, customHandler CustomHandler) {
	exe := os.Args[0]
	if r := recover(); r != nil {
		if err != nil {
			logging.Error("Error occurred but got trumped by panic, original error: %s", errs.JoinMessage(err))
		}

		logging.Error("%s Panic: %v", exe, r)
		logging.Debug("Stack: %s", string(debug.Stack()))

		err = errs.New(fmt.Sprintf(`An unexpected error occurred.
Check the error log for more information.
Your error log is located at: %s`, logging.FilePath()))
		customHandler(err)
		return
	}
	if err != nil {
		logging.Error(exe + " error: " + errs.JoinMessage(err))
		customHandler(err)
	}
}
