package runbits

// Progress bar design
//
import (
	"io"
	"os"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runbits/changesummary"
	"github.com/ActiveState/cli/internal/runbits/progressbar"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/events"
	rtEvents "github.com/ActiveState/cli/pkg/platform/runtime/setup/events"
)

type ProgressDigester interface {
	events.ProgressDigester
	Close()
}

type RuntimeMessageHandler struct {
	progress ProgressDigester
	summary  events.ChangeSummaryDigester
}

func NewRuntimeMessageHandler(progress ProgressDigester, summary events.ChangeSummaryDigester) *RuntimeMessageHandler {
	return &RuntimeMessageHandler{progress, summary}
}

func DefaultRuntimeMessageHandler(out output.Outputer) *RuntimeMessageHandler {
	var w io.Writer = os.Stdout
	if out.Type() != output.PlainFormatName {
		w = nil
	}
	return &RuntimeMessageHandler{
		progress: progressbar.NewRuntimeProgress(w),
		summary:  changesummary.New(out),
	}
}

// HandleEvents prints output based on runtime events received on the events channel
func (rmh *RuntimeMessageHandler) HandleEvents(events <-chan rtEvents.SetupEventer) error {
	// Asynchronous progress digester may need to be closed after
	defer rmh.progress.Close()

	eh := rtEvents.NewRuntimeEventConsumer(rmh.progress, rmh.summary)
	err := eh.Consume(events)
	if err != nil {
		return errs.Wrap(err, "Failed to consume runtime events")
	}

	return nil
}
