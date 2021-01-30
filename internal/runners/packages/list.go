package packages

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/errs"
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
func (l *List) Run(params ListRunParams, nstype model.NamespaceType) error {
	logging.Debug("ExecuteList")

	var commit *strfmt.UUID
	var err error
	switch {
	case params.Commit != "":
		commit, err = targetFromCommit(params.Commit)
		if err != nil {
			return locale.WrapError(err, fmt.Sprintf("%s_err_cannot_obtain_commit", nstype))
		}
	case params.Project != "":
		commit, err = targetFromProject(params.Project)
		if err != nil {
			return locale.WrapError(err, fmt.Sprintf("%s_err_cannot_obtain_commit", nstype))
		}
	default:
		commit, err = targetFromProjectFile()
		if err != nil {
			return locale.WrapError(err, fmt.Sprintf("%s_err_cannot_obtain_commit", nstype))
		}
	}

	checkpoint, err := fetchCheckpoint(commit)
	if err != nil {
		return locale.WrapError(err, fmt.Sprintf("%s_err_cannot_fetch_checkpoint", nstype))
	}

	table := newFilteredRequirementsTable(model.FilterCheckpointPackages(checkpoint), params.Name, nstype)
	table.sortByPkg()

	l.out.Print(table)
	return nil
}

func targetFromCommit(commitOpt string) (*strfmt.UUID, error) {
	if commitOpt == "latest" {
		logging.Debug("latest commit selected")
		proj := project.Get()
		return model.LatestCommitID(proj.Owner(), proj.Name(), proj.BranchName())
	}

	return prepareCommit(commitOpt)
}

func targetFromProject(projectString string) (*strfmt.UUID, error) {
	ns, err := project.ParseNamespace(projectString)
	if err != nil {
		return nil, err
	}

	proj, err := model.FetchProjectByName(ns.Owner, ns.Project)
	if err != nil {
		return nil, err
	}

	for _, branch := range proj.Branches {
		if branch.Default {
			return branch.CommitID, nil
		}
	}

	return nil, locale.NewError("err_package_project_no_commit")
}

func targetFromProjectFile() (*strfmt.UUID, error) {
	logging.Debug("commit from project file")
	proj, err := project.GetSafe()
	if err != nil {
		return nil, err
	}
	commit := proj.CommitID()
	if commit == "" {
		logging.Debug("latest commit used as fallback selection")
		return model.LatestCommitID(proj.Owner(), proj.Name(), proj.BranchName())
	}

	return prepareCommit(commit)
}

func prepareCommit(commit string) (*strfmt.UUID, error) {
	logging.Debug("commit %s selected", commit)
	if ok := strfmt.Default.Validates("uuid", commit); !ok {
		return nil, errs.New("Invalid commit: %s", commit)
	}

	var uuid strfmt.UUID
	if err := uuid.UnmarshalText([]byte(commit)); err != nil {
		return nil, errs.Wrap(err, "UnmarshalText %s failed", commit)
	}

	return &uuid, nil
}

func fetchCheckpoint(commit *strfmt.UUID) (model.Checkpoint, error) {
	if commit == nil {
		logging.Debug("commit id is nil")
		return nil, nil
	}

	checkpoint, _, err := model.FetchCheckpointForCommit(*commit)
	if err != nil && errors.Is(err, model.ErrNoData) {
		return nil, locale.WrapInputError(err, "package_no_data")
	}

	return checkpoint, err
}

func newFilteredRequirementsTable(requirements model.Checkpoint, filter string, nstype model.NamespaceType) *packageTable {
	if requirements == nil {
		logging.Debug("requirements is nil")
		return nil
	}

	rows := make([]packageRow, 0, len(requirements))
	for _, req := range requirements {
		if !strings.Contains(req.Requirement, filter) {
			continue
		}

		if !strings.HasPrefix(req.Namespace, nstype.Prefix()) {
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

	return newTable(rows, locale.T(fmt.Sprintf("%s_list_no_packages", nstype.String())))
}
