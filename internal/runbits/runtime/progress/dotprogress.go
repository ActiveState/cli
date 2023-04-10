package progress

import (
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/events"
)

const dotInterval = time.Second

type DotProgressDigester struct {
	out     output.Outputer
	spinner *output.Spinner
	success bool
}

// NewDotProgressIndicator prints dots at an interval while a runtime is being setup (during solve,
// download, and install steps).
// The primary goal is to indicate to various CI systems (or during non-interactive mode) that
// progress is being made.
func NewDotProgressIndicator(out output.Outputer) *DotProgressDigester {
	return &DotProgressDigester{out: out}
}

func (d *DotProgressDigester) Handle(event events.Eventer) error {
	switch event.(type) {
	case events.Start, events.SolveStart:
		d.spinner = output.StartSpinner(d.out, locale.T("setup_runtime"), time.Second)
	case events.Success:
		d.success = true
	}
	return nil
}

func (d *DotProgressDigester) Close() error {
	if d.spinner == nil {
		return errs.New("spinner not initialized")
	}
	if d.success {
		d.spinner.Stop(locale.T("progress_completed"))
	} else {
		d.spinner.Stop(locale.T("progress_failed"))
	}
	return nil
}
