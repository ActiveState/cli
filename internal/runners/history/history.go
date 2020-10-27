package history

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/pkg/cmdlets/commit"
	gmodel "github.com/ActiveState/cli/pkg/platform/api/graphql/model"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
)

// FailUserNotFound is a failure due to the user not being found
var FailUserNotFound = failures.Type("history.fail.usernotfound")

type History struct{}

func NewHistory() *History {
	return &History{}
}

type HistoryParams struct {
	owner       string
	projectName string
	out         output.Outputer
}

func NewHistoryParams(owner, projectName string, prime primer.Outputer) HistoryParams {
	return HistoryParams{owner, projectName, prime.Output()}
}

func (h *History) Run(params *HistoryParams) error {
	commits, fail := model.CommitHistory(params.owner, params.projectName)
	if fail != nil {
		return fail
	}

	if len(commits) == 0 {
		params.out.Print(locale.Tr("no_commits", project.NewNamespace(params.owner, params.projectName, "").String()))
		return nil
	}

	authorIDs := authorIDsForCommits(commits)
	orgs, fail := model.FetchOrganizationsByIDs(authorIDs)
	if fail != nil {
		return fail
	}

	err := printCommits(params.out, commits, orgs)
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
		authorIDs = append(authorIDs, commit.Author)
	}
	return authorIDs
}
