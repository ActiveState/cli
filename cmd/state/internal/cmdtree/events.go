package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runners/events"
	"github.com/ActiveState/cli/pkg/project"
)

func newEventsCommand(pj *project.Project, output output.Outputer) *captain.Command {
	runner := events.New(pj, output)

	return captain.NewCommand(
		"events",
		locale.Tl("events_description", "Manage project events"),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(cmd *captain.Command, args []string) error {
			return runner.Run()
		})
}
