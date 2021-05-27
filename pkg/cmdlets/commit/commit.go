package commit

import (
	"time"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	gmodel "github.com/ActiveState/cli/pkg/platform/api/graphql/model"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/model"
)

type commitData struct {
	Hash    string   `locale:"hash,[HEADING]Commit[/RESET]"`
	Author  string   `locale:"author,[HEADING]Author[/RESET]"`
	Date    string   `locale:"date,[HEADING]Date[/RESET]"`
	Message string   `locale:"message,[HEADING]Commit Message[/RESET]"`
	Changes []string `locale:"changes,[HEADING]Changes[/RESET]"`
}

func PrintCommit(out output.Outputer, commit *mono_models.Commit, orgs []gmodel.Organization) error {
	data, err := commitDataFromCommit(commit, orgs)
	if err != nil {
		return err
	}
	out.Print(struct {
		commitData `opts:"verticalTable" locale:","`
	}{
		data,
	})

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

	out.Print(struct {
		Data []commitData `opts:"verticalTable" locale:","`
	}{
		Data: data,
	})

	return nil
}

func commitDataFromCommit(commit *mono_models.Commit, orgs []gmodel.Organization) (commitData, error) {
	var username string
	var err error
	if commit.Author != nil && orgs != nil {
		username, err = usernameForID(*commit.Author, orgs)
		if err != nil {
			return commitData{}, locale.WrapError(err, "err_commit_print_username", "Could not determine username for commit author")
		}
	}

	commitData := commitData{
		Hash:    locale.Tl("print_commit_hash", "[ACTIONABLE]{{.V0}}[/RESET]", commit.CommitID.String()),
		Author:  username,
		Changes: formatChanges(commit),
	}

	commitData.Date = commit.AtTime.String()
	dt, err := time.Parse(time.RFC3339, commit.AtTime.String())
	if err != nil {
		logging.Error("Could not parse commit time: %v", err)
	}
	commitData.Date = dt.Format(time.RFC822)

	commitData.Message = locale.Tl("print_commit_no_message", "[DISABLED]Not provided.[/RESET]")
	if commit.Message != "" {
		commitData.Message = commit.Message
	}

	return commitData, nil
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
			),
		)
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
