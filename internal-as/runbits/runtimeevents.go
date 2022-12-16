package runbits

import (
	"io"
	"os"

	"github.com/ActiveState/cli/internal-as/output"
	"github.com/ActiveState/cli/internal-as/runbits/runtime/progress"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/events"
)

func NewRuntimeProgressIndicator(out output.Outputer) events.Handler {
	return newRuntimeProgressIndicator(out)
}

func newRuntimeProgressIndicator(out output.Outputer) events.Handler {
	var w io.Writer = os.Stdout
	if out.Type() != output.PlainFormatName {
		w = nil
	}
	return progress.NewProgressIndicator(w, out)
}
