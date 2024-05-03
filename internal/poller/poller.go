package poller

import (
	"sync"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/runbits/errors"
)

type Poller struct {
	pollFunc      func() (interface{}, error)
	cache         interface{}
	cacheMutex    sync.Mutex
	done          chan struct{}
	errorReported bool
}

func New(interval time.Duration, pollFunc func() (interface{}, error)) *Poller {
	p := &Poller{
		pollFunc: pollFunc,
		done:     make(chan struct{}),
	}
	go p.start(interval)
	return p
}

func (p *Poller) ValueFromCache() interface{} {
	p.cacheMutex.Lock()
	defer p.cacheMutex.Unlock()
	return p.cache
}

func (p *Poller) start(interval time.Duration) {
	timer := time.NewTicker(interval)
	defer timer.Stop()

	p.refresh()

	for {
		select {
		case <-timer.C:
			p.refresh()
		case <-p.done:
			return
		}
	}
}

func (p *Poller) refresh() {
	info, err := p.pollFunc()
	if err != nil {
		if errors.IsReportableError(err) {
			if !p.errorReported {
				multilog.Error("Could not poll: %s", errs.JoinMessage(err))
			} else {
				logging.Debug("Could not poll: %s", errs.JoinMessage(err))
			}
			p.errorReported = true
		}
		return
	}

	p.cacheMutex.Lock()
	p.cache = info
	p.cacheMutex.Unlock()
}

func (p *Poller) Close() {
	close(p.done)
}
