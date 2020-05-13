package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runners/history"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type HistoryOpts struct {
	Namespace string
}

func newHistoryCommand(outputer output.Outputer) *captain.Command {
	initRunner := history.NewHistory()

	opts := HistoryOpts{}
	return captain.NewCommand(
		"history",
		locale.T("history_cmd_description"),
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
				namespace = pj.Namespace()
			}

			nsMeta, fail := project.ParseNamespace(namespace)
			if fail != nil {
				return fail
			}

			params := history.NewHistoryParams(nsMeta.Owner, nsMeta.Project, outputer)
			return initRunner.Run(&params)
		},
	)
}
