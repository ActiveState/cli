package events

import (
	"time"
)

type EventsTimedOutError struct {
}

func (et *EventsTimedOutError) Timeout() bool {
	return true
}

func (et *EventsTimedOutError) Error() string {
	return "timed out waiting for events"
}

func WaitForEvents(t time.Duration, events ...func()) error {
	wg := make(chan struct{})
	go func() {
		for _, event := range events {
			event()
		}
		close(wg)
	}()

	select {
	case <-time.After(t):
		return &EventsTimedOutError{}
	case <-wg:
		return nil
	}
}
