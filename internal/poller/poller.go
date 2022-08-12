package poller

import (
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/multilog"
)

type Poller struct {
	pollFunc func() (interface{}, error)
	cache    interface{}
	done     chan struct{}
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
		multilog.Error("Could not poll %s", errs.JoinMessage(err))
		return
	}

	p.cache = info
}

func (p *Poller) Close() {
	p.done <- struct{}{}
}
