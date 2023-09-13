package analytics

import (
	"github.com/ActiveState/cli/internal/analytics"
	anaConsts "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/pkg/platform/runtime/executors"
)

var isExecutor bool

func init() {
	if isExec, err := executors.IsExecutor(osutils.Executable()); err == nil {
		isExecutor = isExec
	}
}

// Event emits an analytics event with the proper source (State Tool or Executor).
func Event(an analytics.Dispatcher, category, action string, dims ...*dimensions.Values) {
	if isExecutor {
		an.EventWithSource(category, action, anaConsts.SrcExecutor, dims...)
	}
	an.Event(category, action, dims...)
}
