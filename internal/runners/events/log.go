package events

import (
	"regexp"

	"github.com/hpcloud/tail"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/process"
)

type EventLog struct {
	out output.Outputer
}

type EventLogParams struct {
	Follow bool
}

func NewLog(prime primeable) *EventLog {
	return &EventLog{
		prime.Output(),
	}
}

func (e *EventLog) Run(params *EventLogParams) error {
	pid := process.ActivationPID()
	if pid == -1 {
		return locale.NewInputError("err_eventlog_pid", "Could not find parent process ID, make sure you're running this command from inside an activated state (run `state activate` first).")
	}

	filepath := logging.FilePathFor(logging.FileNameFor(int(pid)))
	tailer, err := tail.TailFile(filepath, tail.Config{Follow: params.Follow})
	if err != nil {
		return locale.WrapError(err, "err_tail_start", "Could not tail logging file at {{.V0}}.", logging.FilePath())
	}

	matcher, err := regexp.Compile(`(?:\s|^)(?:\w+-|)Event:`)
	if err != nil {
		return locale.NewError("err_invalid_rx", "Could not create regex matcher. Please contact support, this should not happen.")
	}

	e.out.Notice(output.Title(locale.Tl("events_log_title", "Showing Events Log")))

	for line := range tailer.Lines {
		if matcher.MatchString(line.Text) {
			e.out.Print(line.Text)
		}
	}

	return nil
}
