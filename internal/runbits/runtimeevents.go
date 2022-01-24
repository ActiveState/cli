package runbits

import (
	"io"
	"os"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runbits/buildlogfile"
	"github.com/ActiveState/cli/internal/runbits/changesummary"
	"github.com/ActiveState/cli/internal/runbits/progressbar"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/events"
)

func DefaultRuntimeEventHandler(out output.Outputer) (*events.RuntimeEventHandler, error) {
	return newRuntimeEventHandler(out, changesummary.New(out))
}

func ActivateRuntimeEventHandler(out output.Outputer) (*events.RuntimeEventHandler, error) {
	return newRuntimeEventHandler(out, changesummary.NewEmpty())
}

func newRuntimeEventHandler(out output.Outputer, changeSummary events.ChangeSummaryDigester) (*events.RuntimeEventHandler, error) {
	var w io.Writer = os.Stdout
	if out.Type() != output.PlainFormatName {
		w = nil
	}
	lc, err := buildlogfile.New(out)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to initialize buildlog file handler")
	}

	return events.NewRuntimeEventHandler(progressbar.NewRuntimeProgress(w), changeSummary, lc), nil
}
