package sighandler

import (
	"os"
	"os/signal"
	"sync"

	"github.com/ActiveState/cli/internal-as/errs"
)

type signalStack struct {
	stack []signalStacker
	mu    sync.Mutex
}

func (st *signalStack) Push(s signalStacker) {
	st.mu.Lock()
	defer st.mu.Unlock()

	st.stack = append(st.stack, s)
}

func (st *signalStack) Pop() {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.stack = st.stack[:len(st.stack)-1]
}

func (st *signalStack) Current() signalStacker {
	st.mu.Lock()
	defer st.mu.Unlock()
	if len(st.stack) == 0 {
		return nil
	}

	return st.stack[len(st.stack)-1]
}

// signalStack maintains a stack of signal handlers only the most recent one is active
var stack signalStack

// Push adds a signal handler to the stack of signal handler
// It stops all signal handlers that are lower in the stack and makes the current one the only active one.
func Push(s signalStacker) {
	// stop currently active signal-handler
	cur := stack.Current()
	if cur != nil {
		cur.Pause()
	}

	stack.Push(s)

	s.Resume()
}

// Pop stops and closes the currently active signal handler and removes it from the stack
// The next signal handler in the stack is resumed
func Pop() error {
	cur := stack.Current()
	if cur == nil {
		return errs.New("signal stack has size zero.")
	}
	err := cur.Close()
	if err != nil {
		return err
	}
	stack.Pop()

	cur = stack.Current()
	if cur != nil {
		cur.Resume()
	}
	return nil
}

type signalStacker interface {
	Pause()
	Resume()
	Close() error
}

type sigHandler struct {
	signals []os.Signal
	sigCh   chan (os.Signal)
}

func (sh *sigHandler) Pause() {
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
