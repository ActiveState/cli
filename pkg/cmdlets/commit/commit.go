package commit

import (
	"fmt"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	gmodel "github.com/ActiveState/cli/pkg/platform/api/graphql/model"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/go-openapi/strfmt"
)

type commitOutput struct {
	Hash    string   `locale:"hash,[HEADING]Commit[/RESET]" json:"hash"`
	Author  string   `locale:"author,[HEADING]Author[/RESET]" json:"author"`
	Date    string   `locale:"date,[HEADING]Date[/RESET]" json:"date"`
	Message string   `locale:"message,[HEADING]Message[/RESET]" json:"message"`
	Changes []string `locale:"changes,[HEADING]Changes[/RESET]" json:"changes"`
}

func (o *commitOutput) MarshalOutput(format output.Format) interface{} {
	return struct {
		Data commitOutput `opts:"verticalTable" locale:","`
	}{
		Data: *o,
	}
}

func (o *commitOutput) MarshalStructured(format output.Format) interface{} {
	return o
}

func PrintCommit(out output.Outputer, commit *mono_models.Commit, orgs []gmodel.Organization) error {
	data, err := newCommitOutput(commit, orgs, false)
	if err != nil {
		return err
	}
	out.Print(data)
	return nil
}

func newCommitOutput(commit *mono_models.Commit, orgs []gmodel.Organization, isLocal bool) (*commitOutput, error) {
	var localTxt string
	if isLocal {
		localTxt = locale.Tl("commit_display_local", "[NOTICE] (local)[/RESET]")
	}

	var username string
	var err error
	if commit.Author != nil && orgs != nil {
		username = usernameForID(*commit.Author, orgs)
	}

	commitOutput := &commitOutput{
		Hash:    locale.Tl("print_commit_hash", "[ACTIONABLE]{{.V0}}[/RESET]{{.V1}}", commit.CommitID.String(), localTxt),
		Author:  username,
		Changes: FormatChanges(commit),
	}

	commitOutput.Date = commit.AtTime.String()
	dt, err := time.Parse(time.RFC3339, commit.AtTime.String())
	if err != nil {
		multilog.Error("Could not parse commit time: %v", err)
	}
	commitOutput.Date = dt.Format(time.RFC822)

	commitOutput.Message = locale.Tl("print_commit_no_message", "[DISABLED]Not provided.[/RESET]")
	if commit.Message != "" {
		commitOutput.Message = commit.Message
	}

	return commitOutput, nil
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

func usernameForID(id strfmt.UUID, orgs []gmodel.Organization) string {
	for _, org := range orgs {
		if org.ID == id {
			if org.DisplayName != "" {
				return org.DisplayName
			}
			return org.URLName
		}
	}

	placeholder := locale.Tl("deleted_username", "<deleted>")
	logging.Debug("Could not determine username for commit author '%s'. Using placeholder value '%s'.", id, placeholder)
	return placeholder
}

type commitsOutput []commitOutput

func newCommitsOutput(commits []*mono_models.Commit, orgs []gmodel.Organization, lastRemoteID *strfmt.UUID) (*commitsOutput, error) {
	data := make(commitsOutput, 0, len(commits))
	isLocal := true // recent (and, therefore, local) commits are first

	for _, c := range commits {
		if isLocal && lastRemoteID != nil && c.CommitID == *lastRemoteID {
			isLocal = false
		}

		d, err := newCommitOutput(c, orgs, isLocal)
		if err != nil {
			return nil, err
		}
		data = append(data, *d)
	}

	return &data, nil
}

func (o *commitsOutput) MarshalOutput(format output.Format) interface{} {
	return struct {
		Data commitsOutput `opts:"verticalTable" locale:","`
	}{
		Data: *o,
	}
}

func (o *commitsOutput) MarshalStructured(format output.Format) interface{} {
	return o
}

func PrintCommits(out output.Outputer, commits []*mono_models.Commit, orgs []gmodel.Organization, lastRemoteID *strfmt.UUID) error {
	data, err := newCommitsOutput(commits, orgs, lastRemoteID)
	if err != nil {
		return errs.Wrap(err, "Unable to fetch commit data")
	}
	out.Print(data)
	return nil
}
