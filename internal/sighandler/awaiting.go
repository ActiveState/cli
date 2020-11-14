package sighandler

import (
	"context"
	"fmt"
	"os"
)

var _ signalStacker = &AwaitingSigHandler{}

// SignalError is returned by the AwaitingSigHandler if an interrupt signal was received
type SignalError struct {
	sig os.Signal
}

// Error returns an message about the received event
func (se SignalError) Error() string {
	return fmt.Sprintf("caught a signal: %v", se.sig.String())
}

// Signal returns the received signal
func (se SignalError) Signal() os.Signal {
	return se.Signal()
}

// NewAwaitingSigHandler constructs a signal handler awaiting a function to return
func NewAwaitingSigHandler(signals ...os.Signal) *AwaitingSigHandler {
	return new(signals...)
}

type AwaitingSigHandler = sigHandler

// Close stops the signal handler
func (as *AwaitingSigHandler) Close() error {
	as.Stop()
	return nil
}

// WaitForFunc waits for `f` to return, unless a signal on the sigCh is received.  In that case, we return a SignalError.
func (as *AwaitingSigHandler) WaitForFunc(f func() error) error {
	errCh := make(chan error, 0)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	as.Resume()

	go func() {
		defer close(errCh)
		select {
		case errCh <- f():
		case <-ctx.Done():
		}
	}()

	select {
	case sig := <-as.sigCh:
		return SignalError{sig}
	case err := <-errCh:
		return err
	}
}
