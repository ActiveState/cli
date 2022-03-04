package singlethread

import (
	"fmt"
	"sync"
)

type callback struct {
	funcToCall func() error
	funcResult chan (error)
}

type Thread struct {
	callback chan (callback)
	closed   bool
	mutex    *sync.Mutex
}

func New() *Thread {
	t := &Thread{
		make(chan (callback)),
		false,
		&sync.Mutex{},
	}
	go t.run()
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
