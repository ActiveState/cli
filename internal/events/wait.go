package events

import (
	"time"
)

func WaitForEvents(t time.Duration, events ...func()) {
	wg := make(chan struct{})
	go func() {
		for _, event := range events {
			event()
		}
		close(wg)
	}()

	select {
	case <-time.After(t):
		return
	case <-wg:
		return
	}
}
