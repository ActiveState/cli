package workerpool

import (
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/gammazero/workerpool"
)

// WorkerPool is a wrapper around workerpool.WorkerPool to provide it with some much needed improvements. These are:
// 1. allow for workers to return errors so we don't need to introduce channels to all code using workerpools.
// 2. catch panics inside workers and return them as errors.
type WorkerPool struct {
	size           int
	inner          *workerpool.WorkerPool
	queue          []func() error
	errors         chan error
	errorsOccurred bool
}

func New(maxWorkers int) *WorkerPool {
	return &WorkerPool{
		size:   maxWorkers,
		inner:  workerpool.New(maxWorkers),
		errors: make(chan error),
	}
}

func (wp *WorkerPool) Submit(fn func() error) {
	wp.queue = append(wp.queue, fn)
}

// runQueue will submit the queue of functions to the underlying workerpool library. The reason we do it this way is so
// we have control over what jobs are running, which is in turn important for us to be able to error out as soon as
// possible when an error occurs.
func (wp *WorkerPool) runQueue() {
	n := 0
	for _, fn := range wp.queue {
		if wp.errorsOccurred {
			// No point to keep going if errors have occurred, we want to raise these errors asap.
			break
		}

		wp.inner.Submit(func() {
			defer func() {
				if p := recover(); p != nil {
					wp.errorsOccurred = true
					wp.errors <- errs.New("panic inside workerpool: %v", p)
				}
			}()
			err := fn()
			if err != nil {
				wp.errorsOccurred = true
				wp.errors <- err
			}
		})

		// Give some breathing room for errors to bubble up so we're not running a bunch of jobs we know will
		// result in a failure anyway.
		// The sleep would only cause a slowdown if the previous batch of jobs finished in under the time of the sleep,
		// which is unlikely unless they threw an error.
		if n == wp.size {
			n = 0
			time.Sleep(time.Millisecond * 100)
		}
	}
}

func (wp *WorkerPool) Wait() error {
	wp.runQueue()

	go func() {
		wp.inner.StopWait()
		close(wp.errors)
	}()

	var rerr error
	for err := range wp.errors {
		if rerr == nil {
			rerr = errs.New("workerpool error")
		}
		rerr = errs.Pack(rerr, err)
	}

	return rerr
}
