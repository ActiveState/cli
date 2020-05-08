package events

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/project"
)

type Events struct {
	project *project.Project
	out     output.Outputer
}

func New(pj *project.Project, out output.Outputer) *Events {
	return &Events{
		pj,
		out,
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

	e.out.Notice(locale.Tl("events_listing", "Listing configured events"))

	rows := []Event{}
	for _, event := range e.project.Events() {
		rows = append(rows, Event{
			event.Name(),
			event.Value(),
		})
	}

	if len(rows) == 0 {
		e.out.Print(locale.Tl("events_empty", "No events found for the current project"))
		return nil
	}

	e.out.Print(rows)
	return nil
}
