package sighandler

import (
	"context"
	"os"
)

// BackgroundSigHandler listens for signals in the background
type BackgroundSigHandler struct {
	*sigHandler
	cancel func()
}

// NewBackgroundSignalHandler constructs a signal handler that processes signals in the background until stopped or closed
func NewBackgroundSignalHandler(callback func(os.Signal), signals ...os.Signal) *BackgroundSigHandler {
	ctx, cancel := context.WithCancel(context.Background())
	bs := &BackgroundSigHandler{
		new(signals...),
		cancel,
	}

	go func() {
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
	bs.Stop()
	bs.cancel()
	return nil
}
