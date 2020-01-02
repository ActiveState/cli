package pkg

import (
	"github.com/go-openapi/strfmt"
	"github.com/spf13/cobra"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// ListCommand is the `package list` command struct
var ListCommand = &commands.Command{
	Name:        "list",
	Description: "package_list_description",
	Flags: []*commands.Flag{
		&commands.Flag{
			Name:        "commit",
			Description: "package_list_flag_commit_description",
			Type:        commands.TypeString,
			StringVar:   &ListFlags.Commit,
		},
	},
}

// ListFlags holds the list-related flag values passed through the command line
var ListFlags struct {
	Commit string
}

// ExecuteList lists the current packages in a project
func ExecuteList(cmd *cobra.Command, allArgs []string) {
	logging.Debug("ExecuteList")

	commit, fail := targetedCommit(ListFlags.Commit)
	if fail != nil {
		failures.Handle(fail, locale.T("package_err_cannot_obtain_commit"))
		return
	}

	checkpoint, fail := fetchCheckpoint(commit)
	if fail != nil {
		failures.Handle(fail, locale.T("package_err_cannot_fetch_checkpoint"))
		return
	}
	if len(checkpoint) == 0 {
		print.Line(locale.T("package_no_packages"))
		return
	}

	table := newRequirementsTable(checkpoint)
	sortByFirstCol(table.data)

	print.Line(table.output())
}

func targetedCommit(commitOpt string) (*strfmt.UUID, *failures.Failure) {
	if commitOpt == "latest" {
		logging.Debug("latest commit selected")
		proj := project.Get()
		return model.LatestCommitID(proj.Owner(), proj.Name())
	}

	if commitOpt == "" {
		proj, fail := project.GetSafe()
		if fail != nil {
			return nil, fail
		}
		commitOpt = proj.CommitID()

		if commitOpt == "" {
			logging.Debug("latest commit used as fallback selection")
			return model.LatestCommitID(proj.Owner(), proj.Name())
		}
	}

	logging.Debug("commit %s selected", commitOpt)
	if ok := strfmt.Default.Validates("uuid", commitOpt); !ok {
		return nil, failures.FailMarshal.New(locale.T("invalid_uuid_val"))
	}

	var uuid strfmt.UUID
	if err := uuid.UnmarshalText([]byte(commitOpt)); err != nil {
		return nil, failures.FailMarshal.Wrap(err)
	}

	return &uuid, nil
}

func fetchCheckpoint(commit *strfmt.UUID) (model.Checkpoint, *failures.Failure) {
	if commit == nil {
		logging.Debug("commit id is nil")
		return nil, nil
	}

	checkpoint, _, fail := model.FetchCheckpointForCommit(*commit)
	if fail != nil && fail.Type.Matches(model.FailNoData) {
		return nil, model.FailNoData.New(locale.T("package_no_data"))
	}

	return model.FilterCheckpointPackages(checkpoint), fail
}

func newRequirementsTable(requirements model.Checkpoint) *table {
	if requirements == nil {
		logging.Debug("requirements is nil")
		return nil
	}

	headers := []string{
		locale.T("package_name"),
		locale.T("package_version"),
	}

	rows := make([][]string, 0, len(requirements))
	for _, req := range requirements {
		row := []string{
			req.Requirement,
			req.VersionConstraint,
		}
		rows = append(rows, row)
	}

	return newTable(headers, rows)
}
