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

	timer := time.NewTicker(interval)
	defer timer.Stop()

	done := make(chan struct{})

	go func() {
		i := 0
		for {
			select {
			case <-timer.C:
				i++
				v, ok := p.ValueFromCache().(int)
				if !ok {
					t.Logf("expected int, got %T", v)
					t.Fail()
				}

				if v != i {
					t.Logf("expected %d, got %d", i, v)
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
