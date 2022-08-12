package poller

import (
	"testing"
	"time"
)

func TestPoller(t *testing.T) {
	x := 0
	interval := time.Millisecond * 50
	p := New(interval, func() (interface{}, error) {
		defer func() { x++ }()
		return x, nil
	})
	defer p.Close()

	time.Sleep(time.Millisecond * 10)

	timer := time.NewTicker(interval * 2)
	defer timer.Stop()

	done := make(chan struct{})

	go func() {
		last := -1
		for {
			select {
			case <-timer.C:
				v, ok := p.ValueFromCache().(int)
				if !ok {
					t.Logf("expected int, got %T", v)
					t.Fail()
				}

				if v <= last {
					t.Logf("expected %d to have incremented since last run", v)
					t.Fail()
				}
			case <-done:
				return
			}
		}
	}()

	time.Sleep(time.Second)
	done <- struct{}{}
}
