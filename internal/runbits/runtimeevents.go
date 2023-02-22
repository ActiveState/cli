package runbits

import (
	"io"
	"os"

	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runbits/runtime/progress"
)

func NewRuntimeProgressIndicator(out output.Outputer) *progress.ProgressDigester {
	var w io.Writer = os.Stdout
	if out.Type() != output.PlainFormatName {
		w = nil
	}
	return progress.NewProgressIndicator(w, out)
}
