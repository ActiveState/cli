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
	primer.Outputer
	primer.Projecter
	primer.Configurer
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

	l.out.Print(
		locale.Tl(
			"branch_info",
			"\nBranches allow you to create runtimes with different packages sets depending on your use case. Here are the branches in your current project.\n",
		),
	)

	project, err := model.FetchProjectByName(l.project.Owner(), l.project.Name())
	if err != nil {
		return locale.WrapError(err, "err_fetch_project", "", l.project.Namespace().String())
	}

	tree := NewBranchTree()
	tree.AddLocalBranch(l.project.BranchName())
	tree.AddBranchFormatting("[NOTICE]%s[/RESET]")
	tree.AddLocalBranchFormatting("[ACTIONABLE]%s[/RESET] [DISABLED](Current)[/RESET]")
	tree.BuildFromBranches(project.Branches)
	l.out.Print(tree.String())

	return nil
}
