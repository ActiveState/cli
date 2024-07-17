package runtime

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/runtime/events"
)

func (s *setup) fireEvent(ev events.Event) error {
	for _, h := range s.opts.EventHandlers {
		if err := h(ev); err != nil {
			return errs.Wrap(err, "Event handler failed")
		}
	}
	return nil
}
