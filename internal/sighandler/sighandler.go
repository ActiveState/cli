package sighandler

import (
	"os"
	"os/signal"

	"github.com/ActiveState/cli/internal/errs"
)

// signalStack maintains a stack of signal handlers only the most recent one is active
var signalStack []signalStacker

// Push adds a signal handler to the stack of signal handler
// It stops all signal handlers that are lower in the stack and makes the current one the only active one.
func Push(s signalStacker) {
	// stop currently active signal-handler
	if len(signalStack) > 0 {
		signalStack[len(signalStack)-1].Stop()
	}
	// add new signal-handler
	signalStack = append(signalStack, s)

	s.Resume()
}

// Pop stops and closes the currently active signal handler and removes it from the stack
// The next signal handler in the stack is resumed
func Pop() error {
	if len(signalStack) == 0 {
		return errs.New("signal stack has size zero.")
	}
	err := signalStack[len(signalStack)-1].Close()
	if err != nil {
		return err
	}
	signalStack = signalStack[:len(signalStack)-1]
	if len(signalStack) > 0 {
		signalStack[len(signalStack)-1].Resume()
	}
	return nil
}

type signalStacker interface {
	Stop()
	Resume()
	Close() error
}

type resumeableSignaler interface {
	Stop()
	Resume()
	Signal() <-chan os.Signal
}

var _ signalStacker = &BackgroundSigHandler{}

type sigHandler struct {
	signals []os.Signal
	sigCh   chan (os.Signal)
}

func (sh *sigHandler) Stop() {
	signal.Stop(sh.sigCh)
}

func (sh *sigHandler) Resume() {
	signal.Notify(sh.sigCh, sh.signals...)
}

func new(signals ...os.Signal) *sigHandler {
	sigCh := make(chan os.Signal)
	return &sigHandler{
		signals, sigCh,
	}
}
