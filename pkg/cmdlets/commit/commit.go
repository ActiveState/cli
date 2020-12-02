package commit

import (
	"strings"
	"time"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	gmodel "github.com/ActiveState/cli/pkg/platform/api/graphql/model"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/model"
)

type commitData struct {
	Hash    string   `locale:"hash,Commit"`
	Author  string   `locale:"author,Author"`
	Date    string   `locale:"date,Date"`
	Message string   `locale:"message,Commit Message"`
	Changes []string `locale:"changes,Changes"`
}

func PrintCommit(out output.Outputer, commit *mono_models.Commit, orgs []gmodel.Organization) error {
	data, err := commitDataFromCommit(commit, orgs)
	if err != nil {
		return err
	}
	out.Print(data)

	return nil
}

func PrintCommits(out output.Outputer, commits []*mono_models.Commit, orgs []gmodel.Organization) error {
	var data []commitData
	for _, c := range commits {
		d, err := commitDataFromCommit(c, orgs)
		if err != nil {
			return err
		}
		data = append(data, d)
	}
	out.Print(data)

	return nil
}

func commitDataFromCommit(commit *mono_models.Commit, orgs []gmodel.Organization) (commitData, error) {
	username, err := usernameForID(commit.Author, orgs)
	if err != nil {
		return commitData{}, locale.WrapError(err, "err_commit_print_username", "Could not determine username for commit author")
	}

	return commitData{
		Hash:    commit.CommitID.String(),
		Author:  username,
		Date:    time.Time(commit.Added).Format(constants.DateTimeFormatUser),
		Message: commit.Message,
		Changes: formatChanges(commit),
	}, nil
}

func shortHash(commitID string) string {
	split := strings.Split(commitID, "-")
	if len(split) == 0 {
		return ""
	}
	return split[0]
}

func formatChanges(commit *mono_models.Commit) []string {
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

	return results
}

func usernameForID(id strfmt.UUID, orgs []gmodel.Organization) (string, error) {
	for _, org := range orgs {
		if org.ID == id {
			if org.DisplayName != "" {
				return org.DisplayName, nil
			}
			return org.URLName, nil
		}
	}

	return "", locale.NewError("err_user_not_found", id.String())
}
