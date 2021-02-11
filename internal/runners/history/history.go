package history

import (
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/pkg/cmdlets/commit"
	gmodel "github.com/ActiveState/cli/pkg/platform/api/graphql/model"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

type primeable interface {
	primer.Projecter
	primer.Outputer
}

type History struct {
	project *project.Project
	out     output.Outputer
}

func NewHistory(prime primeable) *History {
	return &History{
		prime.Project(),
		prime.Output(),
	}
}

type HistoryParams struct {
	Namespace string
}

func (h *History) Run(params *HistoryParams) error {
	var commits []*mono_models.Commit
	var err error
	if params.Namespace != "" {
		nsMeta, err := project.ParseNamespace(params.Namespace)
		if err != nil {
			return err
		}

		branch, err := model.DefaultBranchForProjectName(nsMeta.Owner, nsMeta.Project)
		if err != nil {
			return errs.Wrap(err, "Could not get default branch")
		}

		commits, err = model.CommitHistory(nsMeta.Owner, nsMeta.Project, branch.Label)
		if err != nil {
			return locale.WrapError(err, "err_commit_history_namespace", "Could not get commit history from provided namespace: {{.V0}}", params.Namespace)
		}
	} else {
		if h.project == nil {
			return locale.NewInputError("err_history_no_project", "A namespace was not provided and a project could not be found. Please use a project namespace or run this command in a project directory")
		}
		commits, err = model.CommitHistoryFromID(h.project.CommitUUID())
		if err != nil {
			return locale.WrapError(err, "err_commit_hisotry_commit_id", "Could not get commit history from commit ID.")
		}
	}

	if len(commits) == 0 {
		h.out.Print(locale.Tr("no_commits", h.project.Namespace().String()))
		return nil
	}

	authorIDs := authorIDsForCommits(commits)
	orgs, err := model.FetchOrganizationsByIDs(authorIDs)
	if err != nil {
		return err
	}

	h.out.Print(locale.Tl("history_recent_changes", "Here are the most recent changes made to this project.\n"))
	err = commit.PrintCommits(h.out, commits, orgs)
	if err != nil {
		return locale.WrapError(err, "err_history_print_commits", "Could not print commit history")
	}

	return nil
}

func printCommits(out output.Outputer, commits []*mono_models.Commit, orgs []gmodel.Organization) error {
	for _, c := range commits {
		err := commit.PrintCommit(out, c, orgs)
		if err != nil {
			return locale.WrapError(err, "err_history_print", "Encounter error printing commit history")
		}
	}

	return nil
}

func authorIDsForCommits(commits []*mono_models.Commit) []strfmt.UUID {
	authorIDs := []strfmt.UUID{}
	for _, commit := range commits {
		if commit.Author != nil {
			authorIDs = append(authorIDs, *commit.Author)
		}
	}
	return authorIDs
}
