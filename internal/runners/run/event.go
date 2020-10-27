package run

import (
	"fmt"

	"github.com/ActiveState/cli/pkg/project"
)

type Event struct {
	Events []*project.Event
	Type   project.EventType
}

func NewEvent(events []*project.Event, t project.EventType) (*Event, error) {
	return &Event{nil, t}, nil
}

func (es *Event) Run() error {
	fmt.Println(es.Type)
	return nil
}
