package runtime

import (
	"io"
	"os"

	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runbits/runtime/progress"
	"github.com/ActiveState/cli/pkg/runtime/events"
)

func NewRuntimeProgressIndicator(out output.Outputer) events.Handler {
	var w io.Writer = os.Stdout
	if out.Type() != output.PlainFormatName {
		w = nil
	}
	if out.Config().Interactive {
		return progress.NewProgressIndicator(w, out)
	}
	return progress.NewDotProgressIndicator(out)
}
