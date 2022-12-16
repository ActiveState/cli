package output

import (
	"strings"
	"sync"
	"time"
)

type DotProgress struct {
	msg      string
	out      Outputer
	stop     chan struct{}
	stopped  bool
	mutex    *sync.Mutex
	interval time.Duration
}

var _ Marshaller = &DotProgress{}

func (d *DotProgress) MarshalOutput(f Format) interface{} {
	if f != PlainFormatName {
		d.Stop("")
		return Suppress
	}
	return d.msg
}

func NewDotProgress(out Outputer, msg string, interval time.Duration) *DotProgress {
	d := &DotProgress{msg + "...", out, make(chan struct{}, 1), false, &sync.Mutex{}, interval}
	out.Fprint(out.Config().ErrWriter, d)
	go func() {
		d.ticker()
	}()
	return d
}

func (d *DotProgress) ticker() {
	ticker := time.NewTicker(d.interval)
	for {
		select {
		case <-ticker.C:
			d.out.Fprint(d.out.Config().ErrWriter, ".")
		case <-d.stop:
			return
		}
	}
}

func (d *DotProgress) Stopped() bool {
	return d.stopped
}

func (d *DotProgress) Stop(msg string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.stopped {
		return
	}

	d.stop <- struct{}{}
	d.stopped = true

	if msg != "" {
		d.out.Fprint(d.out.Config().ErrWriter, " "+strings.TrimPrefix(msg, " "))
	}

	d.out.Fprint(d.out.Config().ErrWriter, "\n")
}
