package run

import (
	"fmt"

	"github.com/ActiveState/cli/pkg/project"
)

type Event struct {
	DefinedEvents []*project.Event
	CmdList       string
}

func NewEvent(events []*project.Event, cmdList string) *Event {
	return &Event{
		DefinedEvents: events,
		CmdList:       cmdList,
	}
}

func (e *Event) Run(t project.EventType) error {
	var events []*project.Event
	for _, event := range e.DefinedEvents {
		if event.Name() != string(t) {
			continue
		}

		scopes, err := event.Scope()
		if err != nil {
			return err // TODO: this
		}

		for _, scope := range scopes {
			if scope == e.CmdList {
				events = append(events, event)
			}
		}

	}

	for _, event := range events {
		// run logic
		fmt.Println(event)
	}

	fmt.Println(t)
	return nil
}
