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
	fmt.Println(t)
	return nil
}
