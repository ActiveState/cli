package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/history"
)

func newHistoryCommand(prime *primer.Values) *captain.Command {
	initRunner := history.NewHistory(prime)

	params := history.HistoryParams{}
	return captain.NewCommand(
		"history",
		locale.Tl("history_title", "Viewing Project History"),
		locale.T("history_cmd_description"),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, _ []string) error {
			return initRunner.Run(&params)
		},
	).SetGroup(VCSGroup).SetSupportsStructuredOutput()
}
