package run

import (
	"errors"
	"strings"

	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/pkg/project"
)

type Event struct {
	out      output.Outputer
	proj     *project.Project
	subshell subshell.SubShell
	cmdList  string
}

func NewEvent(p primeable, cmdList string) *Event {
	return &Event{
		out:      p.Output(),
		proj:     p.Project(),
		subshell: p.Subshell(),
		cmdList:  cmdList,
	}
}

func (e *Event) Run(t project.EventType) error {
	var events []*project.Event
	for _, event := range e.proj.Events() {
		if event.Name() != string(t) {
			continue
		}

		scopes, err := event.Scope()
		if err != nil {
			return err // TODO: this
		}

		for _, scope := range scopes {
			if scope == e.cmdList {
				events = append(events, event)
			}
		}

	}

	if len(events) == 0 {
		return nil
	}

	r := &Run{
		out:      e.out,
		proj:     e.proj,
		subshell: e.subshell,
	}

	for _, event := range events {
		val, err := event.Value()
		if err != nil {
			return err // TODO: this
		}

		ss := strings.Split(val, " ")
		if len(ss) == 0 {
			return errors.New("no script defined") // TODO: this
		}

		if err := r.Run(ss[0], ss[1:]); err != nil {
			return err // TODO: this
		}
	}

	return nil
}
