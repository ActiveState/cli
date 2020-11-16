package sighandler

import (
	"context"
	"os"
	"sync"
)

var _ signalStacker = &BackgroundSigHandler{}

// BackgroundSigHandler listens for signals in the background
type BackgroundSigHandler struct {
	*sigHandler
	cancel func()
	wg     sync.WaitGroup
}

// NewBackgroundSignalHandler constructs a signal handler that processes signals in the background until stopped or closed
func NewBackgroundSignalHandler(callback func(os.Signal), signals ...os.Signal) *BackgroundSigHandler {
	ctx, cancel := context.WithCancel(context.Background())
	bs := &BackgroundSigHandler{
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
func (bs *BackgroundSigHandler) Close() error {
	bs.Pause()
	bs.cancel()
	bs.wg.Wait()
	return nil
}
