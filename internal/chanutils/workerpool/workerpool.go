package workerpool

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/gammazero/workerpool"
)

// WorkerPool is a wrapper around workerpool.WorkerPool to provide it with some much needed improvements. These are:
// 1. allow for workers to return errors so we don't need to introduce channels to all code using workerpools.
// 2. catch panics inside workers and return them as errors.
type WorkerPool struct {
	inner  *workerpool.WorkerPool
	errors chan error
}

func New(maxWorkers int) *WorkerPool {
	return &WorkerPool{inner: workerpool.New(maxWorkers)}
}

func (wp *WorkerPool) Submit(fn func() error) {
	wp.inner.Submit(func() {
		defer func() {
			if p := recover(); p != nil {
				wp.errors <- errs.New("panic inside workerpool: %v", p)
			}
		}()
		wp.errors <- fn()
	})
}

func (wp *WorkerPool) Wait() error {
	var rerr error
	go func() {
		for err := range wp.errors {
			if err == nil {
				continue
			}
			if rerr == nil {
				rerr = errs.New("workerpool error")
			}
			rerr = errs.Pack(rerr, err)
		}
	}()
	wp.inner.StopWait()
	close(wp.errors)
	return rerr
}
