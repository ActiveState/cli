package branch

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
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
	primer.CheckoutInfoer
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
		return rationalize.ErrNoProject
	}

	project, err := model.LegacyFetchProjectByName(l.project.Owner(), l.project.Name())
	if err != nil {
		return locale.WrapError(err, "err_fetch_project", "", l.project.Namespace().String())
	}

	l.out.Print(output.Prepare(
		branchTree(project.Branches, l.project.BranchName()),
		project.Branches,
	))

	if len(project.Branches) > 1 {
		l.out.Notice(locale.Tl("branch_switch_notice", "To switch to another branch, run '[ACTIONABLE]state branch switch <name>[/RESET]'."))
	}

	return nil
}
