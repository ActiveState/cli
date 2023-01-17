package runbits

import (
	"io"
	"os"

	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runbits/runtime/progress"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/events"
)

func DefaultRuntimeEventHandler(out output.Outputer) events.Handler {
	return newRuntimeEventHandler(out)
}

func newRuntimeEventHandler(out output.Outputer) events.Handler {
	var w io.Writer = os.Stdout
	if out.Type() != output.PlainFormatName {
		w = nil
	}
	return progress.NewProgressDigester(w, out)
}
