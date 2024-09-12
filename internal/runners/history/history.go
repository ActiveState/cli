package history

import (
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	buildscript_runbit "github.com/ActiveState/cli/internal/runbits/buildscript"
	"github.com/ActiveState/cli/internal/runbits/commit"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

type primeable interface {
	primer.Projecter
	primer.Outputer
	primer.Auther
	primer.Configurer
}

type History struct {
	project *project.Project
	out     output.Outputer
	auth    *authentication.Auth
	cfg     *config.Instance
}

func NewHistory(prime primeable) *History {
	return &History{
		prime.Project(),
		prime.Output(),
		prime.Auth(),
		prime.Config(),
	}
}

type HistoryParams struct {
}

func (h *History) Run(params *HistoryParams) error {
	if h.project == nil {
		return locale.NewInputError("err_history_no_project", "No project found. Please run this command in a project directory")
	}
	h.out.Notice(locale.Tr("operating_message", h.project.NamespaceString(), h.project.Dir()))

	localCommitID, err := buildscript_runbit.CommitID(h.project.Dir(), h.cfg)
	if err != nil {
		return errs.Wrap(err, "Unable to get commit ID")
	}

	if h.project.IsHeadless() {
		return locale.NewInputError("err_history_headless", "Cannot get history for headless project. Please visit {{.V0}} to convert your project and try again.", h.project.URL())
	}

	remoteBranch, err := model.BranchForProjectNameByName(h.project.Owner(), h.project.Name(), h.project.BranchName())
	if err != nil {
		return locale.WrapError(err, "err_history_remote_branch", "Could not get branch by local branch name")
	}

	latestRemoteID, err := model.CommonParent(remoteBranch.CommitID, &localCommitID, h.auth)
	if err != nil {
		return locale.WrapError(err, "err_history_common_parent", "Could not determine common parent commit")
	}

	commits, err := model.CommitHistoryFromID(localCommitID, h.auth)
	if err != nil {
		return locale.WrapError(err, "err_commit_history_commit_id", "Could not get commit history from commit ID.")
	}

	if len(commits) == 0 {
		h.out.Print(output.Prepare(locale.Tr("no_commits", h.project.Namespace().String()), []byte("[]")))
		return nil
	}

	authorIDs := authorIDsForCommits(commits)
	orgs, err := model.FetchOrganizationsByIDs(authorIDs, h.auth)
	if err != nil {
		return err
	}

	h.out.Notice(locale.Tl("history_recent_changes", "Here are the most recent changes made to this project.\n"))
	err = commit.PrintCommits(h.out, commits, orgs, latestRemoteID)
	if err != nil {
		return locale.WrapError(err, "err_history_print_commits", "Could not print commit history")
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
