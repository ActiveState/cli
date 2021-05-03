package events

import (
	"github.com/ActiveState/cli/internal/errs"
)

type RuntimeEventHandler struct {
	progress ProgressDigester
	summary  ChangeSummaryDigester
}

func NewRuntimeEventHandler(progress ProgressDigester, summary ChangeSummaryDigester) *RuntimeEventHandler {
	return &RuntimeEventHandler{progress, summary}
}

// WaitForAllEvents prints output based on runtime events received on the events channel
func (rmh *RuntimeEventHandler) WaitForAllEvents(events <-chan SetupEventer) error {
	// Asynchronous progress digester may need to be closed after
	defer rmh.progress.Close()

	eh := NewRuntimeEventConsumer(rmh.progress, rmh.summary)
	err := eh.Consume(events)
	if err != nil {
		return errs.Wrap(err, "Failed to consume runtime events")
	}

	return nil
}
