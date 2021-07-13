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

func NewRuntimeEventHandler(terminalProgress ProgressDigester, summary ChangeSummaryDigester) (*RuntimeEventHandler, error) {
	lc, err := buildlogfile.New()
	if err != nil {
		return nil, errs.Wrap(err, "Failed to initialize the build log file writer")
	}

	return &RuntimeEventHandler{terminalProgress, lc, summary}, nil
}

// WaitForAllEvents prints output based on runtime events received on the events channel
func (rmh *RuntimeEventHandler) WaitForAllEvents(events <-chan SetupEventer) error {
	// Asynchronous progress digester may need to be closed after
	defer rmh.terminalProgress.Close()

	prg := NewMultiPlexedProgress(rmh.terminalProgress, rmh.logFileProgress)
	rec := NewTerminalOutputConsumer(prg, rmh.summary)

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

	return errs.AddTips(err, locale.Tl("build_log_file_hint", "View {{.V0}} for details on build errors.", rmh.logFileProgress.Path()))
}
