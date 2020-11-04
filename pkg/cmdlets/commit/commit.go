package commit

import (
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	gmodel "github.com/ActiveState/cli/pkg/platform/api/graphql/model"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/go-openapi/strfmt"
)

func PrintCommit(out output.Outputer, commit *mono_models.Commit, orgs []gmodel.Organization) error {
	username, err := usernameForID(commit.Author, orgs)
	if err != nil {
		return locale.WrapError(err, "err_commit_print_username", "Could not determine username for commit author")
	}

	out.Print("")
	out.Print(locale.Tr("print_commit", commit.CommitID.String()))
	out.Print(locale.Tr("print_commit_author", username))
	out.Print(locale.Tr("print_commit_date", time.Time(commit.Added).Format(constants.DateTimeFormatUser)))
	if commit.Message != "" {
		out.Print(locale.Tr("print_commit_description", commit.Message))
	}
	out.Print("")
	out.Print(formatChanges(commit))

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
