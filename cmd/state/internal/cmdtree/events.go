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

func newEventsLogCommand(prime *primer.Values) *captain.Command {
	runner := events.NewLog(prime)
	params := events.EventLogParams{}

	return captain.NewCommand(
		"log",
		locale.Tl("events_log_description", "View a log of events"),
		[]*captain.Flag{
			{
				Name:        "follow",
				Shorthand:   "f",
				Description: locale.Tl("tail_f_description", "Don't stop when end of file is reached, wait for additional data."),
				Value:       &params.Follow,
			},
		},
		[]*captain.Argument{},
		func(cmd *captain.Command, args []string) error {
			return runner.Run(&params)
		})
}
