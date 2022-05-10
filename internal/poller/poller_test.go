package poller

import (
	"testing"
	"time"
)

func TestPoller(t *testing.T) {
	x := 0
	interval := time.Millisecond * 10
	p := New(interval, func() (interface{}, error) {
		defer func() { x++ }()
		return x, nil
	})
	defer p.Close()

	time.Sleep(time.Millisecond)
	for i := 0; i < 10; i++ {
		v, ok := p.ValueFromCache().(int)
		if !ok {
			t.Fatalf("expected int, got %T", v)
		}

		if v != i {
			t.Fatalf("expected %d, got %d", i, v)
		}

		time.Sleep(interval)
	}
}
