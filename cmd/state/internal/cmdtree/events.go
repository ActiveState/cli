package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/events"
)

func newEventsCommand(prime *primer.Values) *captain.Command {
	runner := events.New(prime)

	return captain.NewCommand(
		"events",
		locale.Tl("events_description", "Manage project events"),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(cmd *captain.Command, args []string) error {
			return runner.Run()
		})
}
