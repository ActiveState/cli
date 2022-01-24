package events

import (
	"fmt"
	"time"

	"github.com/ActiveState/cli/internal/profile"
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
	defer profile.Measure("event:WaitForEvents", time.Now())
	wg := make(chan struct{})
	go func() {
		for n, event := range events {
			func() {
				defer profile.Measure(fmt.Sprintf("event:WaitForEvents:%d", n), time.Now())
				event()
			}()
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

