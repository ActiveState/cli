package branch

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

type primeable interface {
	primer.Auther
	primer.Outputer
	primer.Projecter
	primer.Configurer
	primer.Analyticer
	primer.SvcModeler
}

type List struct {
	out     output.Outputer
	project *project.Project
}

func NewList(prime primeable) *List {
	return &List{
		out:     prime.Output(),
		project: prime.Project(),
	}
}

func (l *List) Run() error {
	logging.Debug("ExecuteList")

	if l.project == nil {
		return locale.NewInputError("err_no_project")
	}

	project, err := model.FetchProjectByName(l.project.Owner(), l.project.Name())
	if err != nil {
		return locale.WrapError(err, "err_fetch_project", "", l.project.Namespace().String())
	}

	l.out.Print(&branchOutput{
		project.Branches,
		l.project.BranchName(),
	})

	return nil
}
