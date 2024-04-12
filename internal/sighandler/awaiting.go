package sighandler

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/ActiveState/cli/internal/runbits/panics"
)

var _ signalStacker = &Awaiting{}

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
	return se.sig
}

// NewAwaitingSigHandler constructs a signal handler awaiting a function to return
func NewAwaitingSigHandler(signals ...os.Signal) *Awaiting {
	return new(signals...)
}

// Awaiting is a signal handler that is active while waiting for a specific function to finish
type Awaiting = sigHandler

// Close stops the signal handler
func (as *Awaiting) Close() error {
	as.Pause()
	return nil
}

// WaitForFunc waits for `f` to return, unless a signal on the sigCh is received.  In that case, we return a SignalError.
func (as *Awaiting) WaitForFunc(f func() error) error {
	errCh := make(chan error)
	as.Resume()

	go func() {
		defer func() { panics.HandlePanics(recover(), debug.Stack()) }()
		defer close(errCh)
		errCh <- f()
	}()

	select {
	case sig := <-as.sigCh:
		return SignalError{sig}
	case err := <-errCh:
		return err
	}
}
