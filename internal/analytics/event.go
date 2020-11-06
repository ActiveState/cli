package analytics

import (
	ga "github.com/ActiveState/go-ogle-analytics"
)

type event struct {
	Category string
	Action   string
	Label    string
	LabelSet bool
	Value    int64
	ValueSet bool
}

func newEvent(category string, action string) *event {
	return &event{
		Category: category,
		Action:   action,
	}
}

func (e *event) setLabel(label string) *event {
	e.Label = label
	e.LabelSet = true
	return e
}

// Specifies the event value. Values must be non-negative.
func (e *event) setValue(value int64) *event {
	e.Value = value
	e.ValueSet = true
	return e
}

func (e *event) send(c *ga.Client) error {
	gae := ga.NewEvent(e.Category, e.Action)

	if e.LabelSet {
		gae.Label(e.Label)
	}

	if e.ValueSet {
		gae.Value(e.Value)
	}

	return c.Send(gae)
}
