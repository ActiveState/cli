package rtstat

import (
	"sync"
)

// SendOnce provides a simple mechanism to ensure that only one "kind" of
// analytics event is reported per instance lifetime.
type SendOnce struct {
	mu  sync.Mutex
	set map[string]struct{}
}

// NewSendOnce sets up and returns a pointer to an instance of SendOnce.
func NewSendOnce() *SendOnce {
	return &SendOnce{
		set: make(map[string]struct{}),
	}
}

// Send calls send for each "kind" of analytics event only if it has not yet
// been sent.
func (so *SendOnce) Send(stat RtStat) {
	key := stat.String()

	so.mu.Lock()
	defer so.mu.Unlock()

	if _, ok := so.set[key]; ok {
		return
	}
	so.set[key] = struct{}{}
	stat.Send()
}
