package rtwatcher

import (
	"os"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/analytics/client/blackhole"
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

func TestWatcher_check(t *testing.T) {
	w := &Watcher{an: blackhole.Client{}, stop: make(chan struct{}), interval: (10 * time.Millisecond)}
	entries := []entry{
		{
			PID:  123,
			Exec: "not-running",
		},
		{
			PID:  os.Getpid(),
			Exec: os.Args[0],
		},
	}
	w.watching = append(w.watching, entries...)
	go w.ticker(w.check)

	time.Sleep(10 * time.Millisecond)
	w.stop <- struct{}{}

	require.Len(t, w.watching, 1, "Not running process should be removed")
}
