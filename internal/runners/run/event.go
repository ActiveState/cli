package run

import (
	"fmt"

	"github.com/ActiveState/cli/pkg/project"
)

type Event struct {
	DefinedEvents []*project.Event
}

func NewEvent(events []*project.Event) *Event {
	return &Event{
		DefinedEvents: events,
	}
}

func (es *Event) Run(args []string, t project.EventType) error {
	// filter events by type (without mutating DefinedEvents)
	// filter events by args
	// run scripts that remain
	fmt.Println(t)
	return nil
}
