package events

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runbits/buildlogfile"
)

type RuntimeEventHandler struct {
	terminalProgress ProgressDigester
	logFileProgress  *buildlogfile.BuildLogFile
	summary          ChangeSummaryDigester
}

func NewRuntimeEventHandler(terminalProgress ProgressDigester, summary ChangeSummaryDigester, logFileProgress *buildlogfile.BuildLogFile) *RuntimeEventHandler {
	return &RuntimeEventHandler{terminalProgress, logFileProgress, summary}
}

// WaitForAllEvents prints output based on runtime events received on the events channel
func (rmh *RuntimeEventHandler) WaitForAllEvents(events <-chan SetupEventer) error {
	// Asynchronous progress digester may need to be closed after
	prg := NewMultiPlexedProgress(rmh.logFileProgress, rmh.terminalProgress)
	rec := NewRuntimeEventConsumer(prg, rmh.summary)
	defer prg.Close()

	var aggErr error
	for ev := range events {
		err := rec.Consume(ev)
		if err != nil {
			aggErr = errs.Wrap(aggErr, "Event handling error in output consumer: %v", err)
		}
	}

	return aggErr
}

func (rmh *RuntimeEventHandler) AddHints(err error) error {
	if err == nil {
		return nil
	}
	if rmh.logFileProgress == nil {
		return nil
	}

	return errs.AddTips(err, locale.Tl("build_log_file_hint", "Check the Build Log to find out more: [ACTIONABLE]{{.V0}}[/RESET]", rmh.logFileProgress.Path()))
}
