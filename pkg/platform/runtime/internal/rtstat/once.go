package rtstat

import (
	"sync"
)

type SendOnce struct {
	mu  sync.Mutex
	set map[string]struct{}
}

func NewSendOnce() *SendOnce {
	return &SendOnce{
		set: make(map[string]struct{}),
	}
}

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
