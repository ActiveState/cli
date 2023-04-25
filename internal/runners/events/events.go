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

type Event struct {
	Event string `locale:"event,Event" json:"event"`
	Value string `locale:"value,Value" json:"value"`
}

type events []Event

func (e *events) MarshalOutput(format output.Format) interface{} {
	if len(*e) == 0 {
		return locale.Tl("events_empty", "No events found for the current project")
	}
	return e
}

func (e *events) MarshalStructured(format output.Format) interface{} {
	return e
}

func (e *Events) Run() error {
	if e.project == nil {
		return locale.NewInputError("err_events_noproject", "You have to be inside a project folder to be able to view its events. Project folders contain an activestate.yaml.")
	}
	e.out.Notice(locale.Tl("operating_message", "", e.project.NamespaceString(), e.project.Dir()))

	rows := make(events, 0)
	for _, event := range e.project.Events() {
		v, err := event.Value()
		if err != nil {
			return locale.NewError("err_events_val", "Could not get value for event: {{.V0}}.", event.Name())
		}
		rows = append(rows, Event{
			event.Name(),
			v,
		})
	}

	e.out.Print(&rows)
	return nil
}
