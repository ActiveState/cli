package pkg

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/bndr/gotabulate"
	"github.com/go-openapi/strfmt"
	"github.com/spf13/cobra"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
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

	proj := project.Get()

	commit, fail := targetedCommit(proj, ListFlags.Commit)
	if fail != nil {
		failures.Handle(fail, locale.T("package_err_cannot_obtain_commit"))
		return
	}

	reqs, fail := fetchRequirements(commit)
	if fail != nil {
		failures.Handle(fail, locale.T("package_err_cannot_fetch_requirements"))
		return
	}

	rows := makeRequirementsRows(reqs)
	sortByFirstCol(rows)
	table := requirementsTable(rows)

	print.Line(table)
}

func targetedCommit(proj *project.Project, commitOpt string) (*strfmt.UUID, *failures.Failure) {
	if commitOpt == "latest" {
		return model.LatestCommitID(proj.Owner(), proj.Name())
	}

	commit := commitOpt
	if commit == "" {
		commit = proj.CommitID()

		if commit == "" {
			return model.LatestCommitID(proj.Owner(), proj.Name())
		}
	}

	if ok := strfmt.Default.Validates("uuid", commit); !ok {
		err := errors.New("invalid uuid value")
		return nil, failures.FailMarshal.Wrap(err)
	}

	var uuid strfmt.UUID
	if err := uuid.UnmarshalText([]byte(commit)); err != nil {
		return nil, failures.FailMarshal.Wrap(err)
	}

	return &uuid, nil
}

func fetchRequirements(commit *strfmt.UUID) (model.OrderRequirements, *failures.Failure) {
	if commit == nil {
		return nil, nil
	}

	return model.FetchOrderRequirementsByCommit(*commit)
}

type versionRequirements = []*inventory_models.V1OrderRequirementsItemsVersionRequirementsItems

func makeRequirementsRows(reqs model.OrderRequirements) [][]string {
	if reqs == nil {
		return nil
	}

	if len(reqs) == 0 {
		return [][]string{}
	}

	filterFn := func(fallback string) func(*string) string {
		return func(s *string) string {
			if s == nil || *s == "" {
				return fallback
			}
			return *s
		}
	}

	expandVrsReqs := func(vrsReqs versionRequirements) string {
		filterEmpty := filterFn("")

		var bldr strings.Builder
		for _, vrsReq := range vrsReqs {
			fmt.Fprintf(&bldr,
				"%6s %-11s",
				filterEmpty(vrsReq.Comparator),
				filterEmpty(vrsReq.Version),
			)
		}
		return bldr.String()
	}

	filterNone := filterFn(locale.T("none"))

	rows := make([][]string, 0, len(reqs)+1)

	headers := []string{
		locale.T("package_name"),
		locale.T("package_version"),
	}
	rows = append(rows, headers)

	for _, req := range reqs {
		row := []string{
			filterNone(req.Feature),
			expandVrsReqs(req.VersionRequirements),
		}
		rows = append(rows, row)
	}

	return rows
}

func requirementsTable(rows [][]string) string {
	if rows == nil {
		return locale.T("package_no_data")
	}

	if len(rows) == 0 {
		return locale.T("package_no_packages")
	}

	t := gotabulate.Create(rows[1:])
	t.SetHeaders(rows[0])
	t.SetAlign("left")

	return t.Render("simple")
}

func sortByFirstCol(ss [][]string) {
	less := func(i, j int) bool {
		return ss[i][0] < ss[j][0]
	}
	sort.Slice(ss, less)
}
