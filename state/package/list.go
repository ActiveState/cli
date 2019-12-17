package pkg

import (
	"sort"
	"strings"

	"github.com/bndr/gotabulate"
	"github.com/go-openapi/strfmt"
	"github.com/spf13/cobra"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

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

	reqsRows := makeRequirementsRows(checkpoint)
	sortByFirstCol(reqsRows.rows)
	table := requirementsTable(reqsRows)

	print.Line(table)
}

func targetedCommit(commitOpt string) (*strfmt.UUID, *failures.Failure) {
	if commitOpt == "latest" {
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
			return model.LatestCommitID(proj.Owner(), proj.Name())
		}
	}

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
		return nil, nil
	}

	checkpoint, _, fail := model.FetchCheckpointForCommit(*commit)

	return model.FilterCheckpointNoPlatformMatch(checkpoint), fail
}

type requirementsRows struct {
	headers []string
	rows    [][]string
}

func makeRequirementsRows(requirements model.Checkpoint) requirementsRows {
	reqsRows := requirementsRows{}

	if requirements == nil {
		return reqsRows
	}

	reqsRows.headers = []string{
		locale.T("package_name"),
		locale.T("package_version"),
	}

	reqsRows.rows = make([][]string, 0, len(requirements))
	for _, req := range requirements {
		row := []string{
			req.Requirement,
			req.VersionConstraint,
		}
		reqsRows.rows = append(reqsRows.rows, row)
	}

	return reqsRows
}

func requirementsTable(reqsRows requirementsRows) string {
	if reqsRows.rows == nil {
		return locale.T("package_no_data")
	}

	if len(reqsRows.rows) == 0 {
		return locale.T("package_no_packages")
	}

	t := gotabulate.Create(reqsRows.rows)
	t.SetHeaders(reqsRows.headers)
	t.SetAlign("left")

	return t.Render("simple")
}

func sortByFirstCol(rows [][]string) {
	less := func(i, j int) bool {
		if strings.ToLower(rows[i][0]) < strings.ToLower(rows[j][0]) {
			return true
		}
		return rows[i][0] < rows[j][0]
	}
	sort.Slice(rows, less)
}
