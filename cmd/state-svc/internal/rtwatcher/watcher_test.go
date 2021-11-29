package rtwatcher

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWatcher_ticker(t *testing.T) {
	w := &Watcher{stop: make(chan struct{}), interval: (10 * time.Millisecond)}
	calls := 0
	go func() {
		time.Sleep(100 * time.Millisecond)
		w.stop <- struct{}{}
	}()
	w.ticker(func() {
		calls++
	})
	require.Equal(t, 10, calls)
}
