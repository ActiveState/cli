package runbits

import (
	"io"
	"os"

	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runbits/changesummary"
	"github.com/ActiveState/cli/internal/runbits/progressbar"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/events"
)

func DefaultRuntimeEventHandler(out output.Outputer) *events.RuntimeEventHandler {
	var w io.Writer = os.Stdout
	if out.Type() != output.PlainFormatName {
		w = nil
	}
	return events.NewRuntimeEventHandler(progressbar.NewRuntimeProgress(w), changesummary.New(out))
}
