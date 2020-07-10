package history

import (
	"strings"
	"time"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	gmodel "github.com/ActiveState/cli/pkg/platform/api/graphql/model"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
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
		params.out.Print(locale.Tr("no_commits", project.Namespace(params.owner, params.projectName)))
		return nil
	}

	authorIDs := authorIDsForCommits(commits)
	orgs, fail := model.FetchOrganizationsByIDs(authorIDs)
	if fail != nil {
		return fail
	}

	fail = printCommits(params.out, commits, orgs)
	if fail != nil {
		return fail
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

func printCommits(out output.Outputer, commits []*mono_models.Commit, orgs []gmodel.Organization) *failures.Failure {
	for _, c := range commits {
		username, fail := usernameForID(c.Author, orgs)
		if fail != nil {
			return fail
		}

		out.Print("")
		out.Print(locale.Tr("history_commit", c.CommitID.String()))
		out.Print(locale.Tr("history_author", username))
		out.Print(locale.Tr("history_date", time.Time(c.Added).Format(constants.DateTimeFormatUser)))
		if c.Message != "" {
			out.Print(locale.Tr("history_description", c.Message))
		}
		out.Print("")
		out.Print(formatChanges(c))
	}

	return nil
}

func formatChanges(commit *mono_models.Commit) string {
	results := []string{}

	for _, change := range commit.Changeset {
		requirement := change.Requirement
		if model.NamespaceMatch(change.Namespace, model.NamespacePlatformMatch) {
			requirement = locale.T("namespace_label_platform")
		}
		if model.NamespaceMatch(change.Namespace, model.NamespacePrePlatformMatch) {
			requirement = locale.T("namespace_label_preplatform")
		}

		results = append(results,
			locale.Tr("change_"+change.Operation,
				requirement, change.VersionConstraint, change.VersionConstraintOld,
			))
	}

	return strings.Join(results, "\n")
}

func usernameForID(id strfmt.UUID, orgs []gmodel.Organization) (string, *failures.Failure) {
	for _, org := range orgs {
		if org.ID == id {
			if org.DisplayName != "" {
				return org.DisplayName, nil
			}
			return org.URLName, nil
		}
	}

	return "", FailUserNotFound.New(locale.Tr("err_user_not_found", id.String()))
}
