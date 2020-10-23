package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/history"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type historyOpts struct {
	Namespace string
}

func newHistoryCommand(prime *primer.Values) *captain.Command {
	initRunner := history.NewHistory()

	opts := historyOpts{}
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
				Value:       &opts.Namespace,
			},
		},
		[]*captain.Argument{},
		func(ccmd *captain.Command, _ []string) error {
			namespace := opts.Namespace
			if namespace == "" {
				pj, fail := project.GetSafe()
				if fail != nil && fail.Type.Matches(projectfile.FailNoProject) {
					return failures.FailUser.New("err_history_namespace")
				}
				if fail != nil {
					return fail
				}
				namespace = pj.Namespace().String()
			}

			nsMeta, fail := project.ParseNamespace(namespace)
			if fail != nil {
				return fail
			}

			params := history.NewHistoryParams(nsMeta.Owner, nsMeta.Project, prime)
			return initRunner.Run(&params)
		},
	).SetGroup(VCSGroup)
}
