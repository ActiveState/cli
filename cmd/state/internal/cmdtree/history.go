package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/history"
)

type historyOpts struct {
	Namespace string
}

func newHistoryCommand(prime *primer.Values) *captain.Command {
	initRunner := history.NewHistory(prime)

	params := history.HistoryParams{}
	return captain.NewCommand(
		"history",
		locale.Tl("history_title", "Viewing Project History"),
		locale.T("history_cmd_description"),
		prime.Output(),
		[]*captain.Flag{
			{
				Name:        "namespace",
				Shorthand:   "",
				Description: locale.T("arg_state_history_namespace_description"),
				Value:       &params.Namespace,
			},
		},
		[]*captain.Argument{},
		func(ccmd *captain.Command, _ []string) error {
			return initRunner.Run(&params)
		},
	).SetGroup(VCSGroup)
}
