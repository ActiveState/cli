package runtime

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/runtime/events"
)

func fireEvent(handlers []events.HandlerFunc, ev events.Event) error {
	for _, h := range handlers {
		if err := h(ev); err != nil {
			return errs.Wrap(err, "Event handler failed")
		}
	}
	return nil
}
