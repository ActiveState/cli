package singlethread

import (
	"fmt"
	"sync"

	"github.com/ActiveState/cli/internal/osutils/stacktrace"
)

type callback struct {
	funcToCall func() error
	funcResult chan (error)
}

type Thread struct {
	callback chan (callback)
	closed   bool
	mutex    *sync.Mutex
	stack    string
}

var history []*Thread

func PrintNotClosedThreads() {
	for _, thread := range history {
		if !thread.closed {
			fmt.Printf("Thread not closed: %s", thread.stack)
		}
	}
}

func New() *Thread {
	t := &Thread{
		make(chan (callback)),
		false,
		&sync.Mutex{},
		stacktrace.Get().String(),
	}
	go t.run()
	history = append(history, t)
	return t
}

func (t *Thread) run() {
	for {
		callback, gotCallback := <-t.callback
		if gotCallback {
			callback.funcResult <- callback.funcToCall()
		} else {
			return
		}
	}
}

func (t *Thread) Run(funcToCall func() error) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.closed {
		return fmt.Errorf("thread is closed")
	}

	callback := callback{funcToCall, make(chan (error))}
	t.callback <- callback
	return <-callback.funcResult
}

func (t *Thread) Close() {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	close(t.callback)
	t.closed = true
}
