package singlethread

import "fmt"

type callback struct {
	cb  func() error
	err chan (error)
}

type Thread struct {
	cbs    chan (callback)
	close  chan (struct{})
	closed bool
}

func New() *Thread {
	t := &Thread{
		make(chan (callback)),
		make(chan (struct{}), 1),
		false,
	}
	go t.run()
	return t
}

func (t *Thread) run() {
	for {
		select {
		case cbs := <-t.cbs:
			cbs.err <- cbs.cb()
		case <-t.close:
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
	t.close <- struct{}{}
	t.closed = true
}
