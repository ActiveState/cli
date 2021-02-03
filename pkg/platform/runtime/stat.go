package runtime

import (
	"sync"

	"github.com/ActiveState/cli/pkg/platform/runtime/internal/rtstat"
)

type sendOnce struct {
	mu  sync.Mutex
	set map[string]struct{}
}

func newSendOnce() *sendOnce {
	return &sendObnce{
		set: make(map[string]struct{}),
	}
}

func (so *sendOnce) send(stat rtstat.RtStat) {
	key := stat.String()

	so.mu.Lock()
	defer so.mu.Unlock()

	if _, ok := so.set[key]; ok {
		return
	}
	so.set[key] = struct{}{}
	stat.Send()
}
