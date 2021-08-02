package commit

import (
	"fmt"
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
	Hash    string   `locale:"hash,[HEADING]Commit[/RESET]"`
	Author  string   `locale:"author,[HEADING]Author[/RESET]"`
	Date    string   `locale:"date,[HEADING]Date[/RESET]"`
	Message string   `locale:"message,[HEADING]Message[/RESET]"`
	Changes []string `locale:"changes,[HEADING]Changes[/RESET]"`
}

func PrintCommit(out output.Outputer, commit *mono_models.Commit, orgs []gmodel.Organization) error {
	data, err := commitDataFromCommit(commit, orgs, false)
	if err != nil {
		return err
	}
	out.Print(struct {
		Data commitData `opts:"verticalTable" locale:","`
	}{
		Data: data,
	})

	return nil
}

var EarliestRemoteID = func() *strfmt.UUID {
	id := strfmt.UUID("earliest")
	return &id
}()

func PrintCommits(out output.Outputer, commits []*mono_models.Commit, orgs []gmodel.Organization, lastRemoteID *strfmt.UUID) error {
	data := make([]commitData, 0, len(commits))
	isLocal := true // recent (and, therefore, local) commits are first

	for _, c := range commits {
		if lastRemoteID != EarliestRemoteID || lastRemoteID == nil || (isLocal && c.CommitID == *lastRemoteID) {
			isLocal = false
		}

		d, err := commitDataFromCommit(c, orgs, isLocal)
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

func commitDataFromCommit(commit *mono_models.Commit, orgs []gmodel.Organization, isLocal bool) (commitData, error) {
	var localTxt string
	if isLocal {
		localTxt = locale.Tl("commit_display_local", "[NOTICE] (local)[/RESET]")
	}

	var username string
	var err error
	if commit.Author != nil && orgs != nil {
		username, err = usernameForID(*commit.Author, orgs)
		if err != nil {
			return commitData{}, locale.WrapError(err, "err_commit_print_username", "Could not determine username for commit author")
		}
	}

	commitData := commitData{
		Hash:    locale.Tl("print_commit_hash", "[ACTIONABLE]{{.V0}}[/RESET]{{.V1}}", commit.CommitID.String(), localTxt),
		Author:  username,
		Changes: FormatChanges(commit),
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

func FormatChanges(commit *mono_models.Commit) []string {
	results := []string{}

	for _, change := range commit.Changeset {
		requirement := change.Requirement
		versionConstraints := formatConstraints(change.VersionConstraints)
		if model.NamespaceMatch(change.Namespace, model.NamespacePlatformMatch) {
			requirement = locale.T("namespace_label_platform")
			versionConstraints = ""
		}
		if model.NamespaceMatch(change.Namespace, model.NamespacePrePlatformMatch) {
			requirement = locale.T("namespace_label_preplatform")
			versionConstraints = ""
		}

		var result string
		switch change.Operation {
		case string(model.OperationAdded):
			result = locale.Tr("change_added", requirement, versionConstraints)
		case string(model.OperationRemoved):
			result = locale.Tr("change_removed", requirement)
		case string(model.OperationUpdated):
			result = locale.Tr("change_updated", requirement, formatConstraints(change.VersionConstraintsOld), versionConstraints)
		}
		results = append(results, result)
	}

	return results
}

func formatConstraints(constraints []*mono_models.Constraint) string {
	if len(constraints) == 0 {
		return locale.Tl("constraint_auto", "Auto")
	}

	var result []string
	for _, constraint := range constraints {
		var comparator string
		switch constraint.Comparator {
		case "eq":
			return constraint.Version
		case "gt":
			comparator = ">"
		case "gte":
			comparator = ">="
		case "lt":
			comparator = "<"
		case "lte":
			comparator = "<="
		case "ne":
			comparator = "!="
		default:
			comparator = "?"
		}
		result = append(result, fmt.Sprintf("%s%s", comparator, constraint.Version))
	}
	return strings.Join(result, ",")
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
