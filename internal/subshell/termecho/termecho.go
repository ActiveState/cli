package termecho

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/multilog"
)

func Off() {
	err := toggle(false)
	if err != nil {
		multilog.Error("Unable to turn off terminal echoing: %v", errs.JoinMessage(err))
	}
}

func On() {
	err := toggle(true)
	if err != nil {
		multilog.Error("Unable to turn off terminal echoing: %v", errs.JoinMessage(err))
	}
}
