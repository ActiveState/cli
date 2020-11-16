package sighandler

import (
	"context"
	"os"
	"sync"
)

var _ signalStacker = &Background{}

// Background listens for signals in the background
type Background struct {
	*sigHandler
	cancel func()
	wg     sync.WaitGroup
}

// NewBackgroundSignalHandler constructs a signal handler that processes signals in the background until stopped or closed
func NewBackgroundSignalHandler(callback func(os.Signal), signals ...os.Signal) *Background {
	ctx, cancel := context.WithCancel(context.Background())
	bs := &Background{
		new(signals...),
		cancel,
		sync.WaitGroup{},
	}

	bs.wg.Add(1)
	go func() {
		defer bs.wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case s := <-bs.sigCh:
				callback(s)
			}
		}
	}()

	return bs
}

// Close cancels the background process
func (bs *Background) Close() error {
	bs.Pause()
	bs.cancel()
	bs.wg.Wait()
	return nil
}
