package events

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/pkg/project"
)

type Events struct {
	project *project.Project
	out     output.Outputer
}

type primeable interface {
	primer.Projecter
	primer.Outputer
	primer.Configurer
}

func New(prime primeable) *Events {
	return &Events{
		prime.Project(),
		prime.Output(),
	}
}

type eventOutput struct {
	Event string `locale:"event,Event" json:"event"`
	Value string `locale:"value,Value" json:"value"`
}

func (e *Events) Run() error {
	if e.project == nil {
		return locale.NewInputError("err_events_noproject", "You have to be inside a project folder to be able to view its events. Project folders contain an activestate.yaml.")
	}
	e.out.Notice(locale.Tr("operating_message", e.project.NamespaceString(), e.project.Dir()))

	events := e.project.Events()
	rows := make([]eventOutput, len(events))
	for i, event := range events {
		v, err := event.Value()
		if err != nil {
			return locale.NewError("err_events_val", "Could not get value for event: {{.V0}}.", event.Name())
		}
		rows[i] = eventOutput{event.Name(), v}
	}

	var plainOutput interface{} = rows
	if len(rows) == 0 {
		plainOutput = locale.Tl("events_empty", "No events found for the current project")
	}
	e.out.Print(output.Prepare(plainOutput, rows))
	return nil
}
