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
	Hash              string               `locale:"hash,[HEADING]Commit[/RESET]" json:"hash"`
	Author            string               `locale:"author,[HEADING]Author[/RESET]" json:"author"`
	Date              string               `locale:"date,[HEADING]Date[/RESET]" json:"date"`
	Revision          string               `locale:"revision,[HEADING]Revision[/RESET]" json:"revision"`
	Message           string               `locale:"message,[HEADING]Message[/RESET]" json:"message"`
	PlainChanges      []string             `locale:"changes,[HEADING]Changes[/RESET]" json:"-"`
	StructuredChanges []*requirementChange `opts:"hidePlain" json:"changes"`
}

type requirementChange struct {
	Operation             string `json:"operation"`
	Requirement           string `json:"requirement"`
	VersionConstraintsOld string `json:"version_constraints_old,omitempty"`
	VersionConstraintsNew string `json:"version_constraints_new,omitempty"`
	Namespace             string `json:"namespace"`
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

	plainChanges, structuredChanges := FormatChanges(commit)

	commitOutput := &commitOutput{
		Hash:              locale.Tl("print_commit_hash", "[ACTIONABLE]{{.V0}}[/RESET]{{.V1}}", commit.CommitID.String(), localTxt),
		Author:            username,
		PlainChanges:      plainChanges,
		StructuredChanges: structuredChanges,
	}

	dt, err := time.Parse(time.RFC3339, commit.Added.String())
	if err != nil {
		multilog.Error("Could not parse commit time: %v", errs.JoinMessage(err))
	}
	commitOutput.Date = dt.Format(time.RFC822)

	dt, err = time.Parse(time.RFC3339, commit.AtTime.String())
	if err != nil {
		multilog.Error("Could not parse revision time: %v", errs.JoinMessage(err))
	}
	commitOutput.Revision = dt.Format(time.RFC822)

	commitOutput.Message = locale.Tl("print_commit_no_message", "[DISABLED]Not provided.[/RESET]")
	if commit.Message != "" {
		commitOutput.Message = commit.Message
	}

	return commitOutput, nil
}

func FormatChanges(commit *mono_models.Commit) ([]string, []*requirementChange) {
	results := []string{}
	requirements := []*requirementChange{}

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

		// This is a temporary fix until we start getting history in the form of build expressions
		// https://activestatef.atlassian.net/browse/DX-2197
		if model.NamespaceMatch(change.Namespace, model.NamespaceBuildFlagsMatch) &&
			(strings.Contains(change.Requirement, "docker") || strings.Contains(change.Requirement, "installer")) {
			requirement = locale.T("namespace_label_packager")
			versionConstraints = ""
		}

		var result, oldConstraints, newConstraints string
		switch change.Operation {
		case string(model.OperationAdded):
			result = locale.Tr("change_added", requirement, versionConstraints, change.Namespace)
			newConstraints = formatConstraints(change.VersionConstraints)
		case string(model.OperationRemoved):
			result = locale.Tr("change_removed", requirement, change.Namespace)
			oldConstraints = formatConstraints(change.VersionConstraintsOld)
		case string(model.OperationUpdated):
			result = locale.Tr("change_updated", requirement, formatConstraints(change.VersionConstraintsOld), versionConstraints, change.Namespace)
			oldConstraints = formatConstraints(change.VersionConstraintsOld)
			newConstraints = formatConstraints(change.VersionConstraints)
		}
		results = append(results, result)

		requirements = append(requirements, &requirementChange{
			Operation:             change.Operation,
			Requirement:           change.Requirement,
			VersionConstraintsOld: oldConstraints,
			VersionConstraintsNew: newConstraints,
			Namespace:             change.Namespace,
		})
	}

	return results, requirements
}

func formatConstraints(constraints []*mono_models.Constraint) string {
	if len(constraints) == 0 {
		return locale.T("constraint_auto")
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
