package packages

import (
	"strings"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// ListRunParams tracks the info required for running List.
type ListRunParams struct {
	Commit  string
	Name    string
	Project string
}

// List manages the listing execution context.
type List struct {
	out output.Outputer
}

// NewList prepares a list execution context for use.
func NewList(prime primer.Outputer) *List {
	return &List{
		out: prime.Output(),
	}
}

// Run executes the list behavior.
func (l *List) Run(params ListRunParams, pt PackageType) error {
	logging.Debug("ExecuteList")

	var commit *strfmt.UUID
	var fail *failures.Failure
	switch {
	case params.Commit != "":
		commit, fail = targetFromCommit(params.Commit)
		if fail != nil {
			return locale.WrapError(fail.ToError(), "package_err_cannot_obtain_commit", pt.String())
		}
	case params.Project != "":
		commit, fail = targetFromProject(params.Project)
		if fail != nil {
			return locale.WrapError(fail.ToError(), "package_err_cannot_obtain_commit", pt.String())
		}
	default:
		commit, fail = targetFromProjectFile()
		if fail != nil {
			return locale.WrapError(fail.ToError(), "package_err_cannot_obtain_commit", pt.String())
		}
	}

	checkpoint, fail := fetchCheckpoint(commit)
	if fail != nil {
		return locale.WrapError(fail.ToError(), "package_err_cannot_fetch_checkpoint", pt.String())
	}

	table := newFilteredRequirementsTable(model.FilterCheckpointPackages(checkpoint), params.Name, pt)
	table.sortByPkg()

	l.out.Print(table)
	return nil
}

func targetFromCommit(commitOpt string) (*strfmt.UUID, *failures.Failure) {
	if commitOpt == "latest" {
		logging.Debug("latest commit selected")
		proj := project.Get()
		return model.LatestCommitID(proj.Owner(), proj.Name())
	}

	return prepareCommit(commitOpt)
}

func targetFromProject(projectString string) (*strfmt.UUID, *failures.Failure) {
	ns, fail := project.ParseNamespace(projectString)
	if fail != nil {
		return nil, fail
	}

	proj, fail := model.FetchProjectByName(ns.Owner, ns.Project)
	if fail != nil {
		return nil, fail
	}

	for _, branch := range proj.Branches {
		if branch.Default {
			return branch.CommitID, nil
		}
	}

	return nil, failures.FailNotFound.New(locale.T("err_package_project_no_commit"))
}

func targetFromProjectFile() (*strfmt.UUID, *failures.Failure) {
	logging.Debug("commit from project file")
	proj, fail := project.GetSafe()
	if fail != nil {
		return nil, fail
	}
	commit := proj.CommitID()
	if commit == "" {
		logging.Debug("latest commit used as fallback selection")
		return model.LatestCommitID(proj.Owner(), proj.Name())
	}

	return prepareCommit(commit)
}

func prepareCommit(commit string) (*strfmt.UUID, *failures.Failure) {
	logging.Debug("commit %s selected", commit)
	if ok := strfmt.Default.Validates("uuid", commit); !ok {
		return nil, failures.FailMarshal.New(locale.T("invalid_uuid_val"))
	}

	var uuid strfmt.UUID
	if err := uuid.UnmarshalText([]byte(commit)); err != nil {
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

	return checkpoint, fail
}

func newFilteredRequirementsTable(requirements model.Checkpoint, filter string, pt PackageType) *packageTable {
	if requirements == nil {
		logging.Debug("requirements is nil")
		return nil
	}

	rows := make([]packageRow, 0, len(requirements))
	for _, req := range requirements {
		if !strings.Contains(req.Requirement, filter) {
			continue
		}

		if !strings.HasPrefix(req.Namespace, string(pt.Namespace())) {
			continue
		}

		versionConstraint := req.VersionConstraint
		if versionConstraint == "" {
			versionConstraint = "Auto"
		}

		row := packageRow{
			req.Requirement,
			versionConstraint,
		}
		rows = append(rows, row)
	}

	return newTable(rows, locale.Tr("package_list_no_packages", pt.String()))
}
