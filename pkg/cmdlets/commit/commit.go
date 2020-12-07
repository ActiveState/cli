package commit

import (
	"strings"
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
	Heading string `locale:"heading,"`
	Data    string `locale:"data,"`
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
	var data [][]commitData
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

func commitDataFromCommit(commit *mono_models.Commit, orgs []gmodel.Organization) ([]commitData, error) {
	var username string
	var err error
	if commit.Author != nil && orgs != nil {
		username, err = usernameForID(*commit.Author, orgs)
		if err != nil {
			return nil, locale.WrapError(err, "err_commit_print_username", "Could not determine username for commit author")
		}
	}

	data := make([]commitData, 0)
	data = append(data, commitData{
		locale.Tl("print_commit_hash_heading", "[HEADING]Commit[/RESET]"),
		locale.Tl("print_commit_hash", "[ACTIONABLE]{{.V0}}[/RESET]", commit.CommitID.String()),
	})

	data = append(data, commitData{
		locale.Tl("print_commit_author_heading", "[HEADING]Author[/RESET]"),
		username,
	})

	commitTime := commit.AtTime.String()
	dt, err := time.Parse(time.RFC3339, commit.AtTime.String())
	if err != nil {
		logging.Error("Could not parse commit time: %v", err)
	}
	commitTime = dt.Format(time.RFC822)
	data = append(data, commitData{
		locale.Tl("print_commit_time_heading", "[HEADING]Date[/RESET]"),
		commitTime,
	})

	message := locale.Tl("print_commit_no_message", "[DISABLED]Not provided.[/RESET]")
	if commit.Message != "" {
		message = commit.Message
	}
	data = append(data, commitData{
		locale.Tl("print_commit_description_heading", "[HEADING]Description[/RESET]"),
		message,
	})

	changes := formatChanges(commit)
	for i, change := range changes {
		switch {
		case strings.Contains(change, "added"):
			change = locale.Tl("print_commit_added_change", "[SUCCESS]+[/RESET] {{.V0}}", change)
		case strings.Contains(change, "removed"):
			change = locale.Tl("print_commit_removed_change", "[SUCCESS]+[/RESET] {{.V0}}", change)
		case strings.Contains(change, "updated"):
			change = locale.Tl("print_commit_updated_change", "[ACTIONABLE]•[/RESET] {{.V0}}", change)
		}
		changes[i] = change
	}
	data = append(data, commitData{
		locale.Tl("print_commit_changes_heading", "[HEADING]Changes[/RESET]"),
		strings.Join(changes, "\n"),
	})

	return data, nil
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
