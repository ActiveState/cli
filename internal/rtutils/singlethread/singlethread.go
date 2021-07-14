package singlethread

import (
	"fmt"
	"sync"
)

type callback struct {
	cb  func() error
	err chan (error)
}

type Thread struct {
	cbs    chan (callback)
	closed bool
}

func New() *Thread {
	t := &Thread{
		make(chan (callback)),
		false,
	}
	go t.run()
	return t
}

func (t *Thread) run() {
	for {
		cbs, more := <-t.cbs
		if more {
			cbs.err <- cbs.cb()
		} else {
			return
		}
	}
}

func (t *Thread) Run(cb func() error) error {
	if t.closed {
		return fmt.Errorf("thread is closed")
	}
	cbs := callback{cb, make(chan (error))}
	t.cbs <- cbs
	return <-cbs.err
}

func (t *Thread) Close() {
	mutex := sync.Mutex{}
	mutex.Lock()
	defer mutex.Unlock()
	
	close(t.cbs)
	t.closed = true
}
