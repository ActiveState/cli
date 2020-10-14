package events

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/table"
	"github.com/ActiveState/cli/pkg/project"
)

type Events struct {
	project *project.Project
	out     output.Outputer
}

type primeable interface {
	primer.Projecter
	primer.Outputer
}

func New(prime primeable) *Events {
	return &Events{
		prime.Project(),
		prime.Output(),
	}
}

type Event struct {
	Event string `locale:"event,Event"`
	Value string `locale:"value,Value" opts:"singleLine"`
}

func (e *Events) Run() error {
	if e.project == nil {
		return locale.NewInputError("err_events_noproject", "You have to be inside a project folder to be able to view its events. Project folders contain an activestate.yaml.")
	}

	rows := []Event{}
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

	table := table.NewTable(rows, locale.Tl("list_events_info", "Here are all the events for your current project"), locale.Tl("events_empty", "No events found for the current project"))
	e.out.Print(table)
	return nil
}
