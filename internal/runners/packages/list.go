package packages

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/runbits/commitmediator"
	runbitsRuntime "github.com/ActiveState/cli/internal/runbits/runtime"
	gqlModel "github.com/ActiveState/cli/pkg/platform/api/graphql/model"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/ActiveState/cli/pkg/platform/runtime/store"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
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
	out       output.Outputer
	project   *project.Project
	analytics analytics.Dispatcher
	svcModel  *model.SvcModel
	auth      *authentication.Auth
}

// NewList prepares a list execution context for use.
func NewList(prime primeable) *List {
	return &List{
		out:       prime.Output(),
		project:   prime.Project(),
		analytics: prime.Analytics(),
		svcModel:  prime.SvcModel(),
		auth:      prime.Auth(),
	}
}

// Run executes the list behavior.
func (l *List) Run(params ListRunParams, nstype model.NamespaceType) error {
	logging.Debug("ExecuteList")

	if l.project != nil && params.Project == "" {
		l.out.Notice(locale.Tr("operating_message", l.project.NamespaceString(), l.project.Dir()))
	}

	var commit *strfmt.UUID
	var err error
	switch {
	case params.Commit != "":
		commit, err = targetFromCommit(params.Commit, l.project)
		if err != nil {
			return locale.WrapError(err, fmt.Sprintf("%s_err_cannot_obtain_commit", nstype))
		}
	case params.Project != "":
		commit, err = targetFromProject(params.Project)
		if err != nil {
			return locale.WrapError(err, fmt.Sprintf("%s_err_cannot_obtain_commit", nstype))
		}
	default:
		commit, err = targetFromProjectFile(l.project)
		if err != nil {
			return locale.WrapError(err, fmt.Sprintf("%s_err_cannot_obtain_commit", nstype))
		}
	}

	checkpoint, err := fetchCheckpoint(commit)
	if err != nil {
		return locale.WrapError(err, fmt.Sprintf("%s_err_cannot_fetch_checkpoint", nstype))
	}

	// Initialize the project's runtime and determine its language if possible.
	// This is used for resolving package version numbers.
	// Note: any errors here are not fatal, and should not be reported to rollbar.
	var rt *runtime.Runtime
	if l.project != nil && params.Project == "" {
		rt, err = runbitsRuntime.NewFromProject(l.project, target.TriggerPackage, l.analytics, l.svcModel, l.out, l.auth)
		if err != nil {
			logging.Error("Unable to initialize runtime for version resolution: %v", errs.JoinMessage(err))
		}
	}
	var ns *model.Namespace
	if language, err := model.LanguageByCommit(*commit); err == nil {
		ns = ptr.To(model.NewNamespacePkgOrBundle(language.Name, nstype))
	} else {
		logging.Error("Unable to get language from project: %v", errs.JoinMessage(err))
	}

	rows := newFilteredRequirementsTable(model.FilterCheckpointNamespace(checkpoint, model.NamespacePackage, model.NamespaceBundle), params.Name, nstype, rt, ns)
	var plainOutput interface{} = rows
	if len(rows) == 0 {
		plainOutput = locale.T(fmt.Sprintf("%s_list_no_packages", nstype.String()))
	}
	l.out.Print(output.Prepare(plainOutput, rows))
	return nil
}

func targetFromCommit(commitOpt string, proj *project.Project) (*strfmt.UUID, error) {
	if commitOpt == "latest" {
		logging.Debug("latest commit selected")
		return model.BranchCommitID(proj.Owner(), proj.Name(), proj.BranchName())
	}

	return prepareCommit(commitOpt)
}

func targetFromProject(projectString string) (*strfmt.UUID, error) {
	ns, err := project.ParseNamespace(projectString)
	if err != nil {
		return nil, err
	}

	branch, err := model.DefaultBranchForProjectName(ns.Owner, ns.Project)
	if err != nil {
		return nil, errs.Wrap(err, "Could not grab default branch for project")
	}

	return branch.CommitID, nil
}

func targetFromProjectFile(proj *project.Project) (*strfmt.UUID, error) {
	logging.Debug("commit from project file")
	if proj == nil {
		return nil, locale.NewInputError("err_no_project")
	}
	commit, err := commitmediator.Get(proj)
	if err != nil {
		return nil, errs.Wrap(err, "Unable to get local commit")
	}
	if commit == "" {
		logging.Debug("latest commit used as fallback selection")
		return model.BranchCommitID(proj.Owner(), proj.Name(), proj.BranchName())
	}

	return prepareCommit(commit.String())
}

func prepareCommit(commit string) (*strfmt.UUID, error) {
	logging.Debug("commit %s selected", commit)
	if ok := strfmt.Default.Validates("uuid", commit); !ok {
		return nil, locale.NewInputError("err_invalid_commit", "Invalid commit: {{.V0}}", commit)
	}

	var uuid strfmt.UUID
	if err := uuid.UnmarshalText([]byte(commit)); err != nil {
		return nil, errs.Wrap(err, "UnmarshalText %s failed", commit)
	}

	return &uuid, nil
}

func fetchCheckpoint(commit *strfmt.UUID) ([]*gqlModel.Requirement, error) {
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

type packageRow struct {
	Pkg     string `json:"package" locale:"package_name,Name"`
	Version string `json:"version" locale:"package_version,Version"`
}

func newFilteredRequirementsTable(requirements []*gqlModel.Requirement, filter string, nstype model.NamespaceType, rt *runtime.Runtime, ns *model.Namespace) []packageRow {
	if requirements == nil {
		logging.Debug("requirements is nil")
		return nil
	}

	// Fetch resolved artifacts list for showing full version numbers.
	// Note: an error here is not fatal.
	var artifacts []artifact.Artifact
	if rt != nil && ns != nil {
		var err error
		artifacts, err = rt.ResolvedArtifacts()
		if !errs.Matches(err, store.ErrNoBuildPlanFile) {
			multilog.Error("Unable to retrieve runtime resolved artifact list: %v", errs.JoinMessage(err))
		}
	}

	rows := make([]packageRow, 0, len(requirements))
	for _, req := range requirements {
		if !strings.Contains(strings.ToLower(req.Requirement), strings.ToLower(filter)) {
			continue
		}

		if !strings.HasPrefix(req.Namespace, nstype.Prefix()) {
			continue
		}

		versionConstraint := req.VersionConstraint
		if versionConstraint == "" {
			versionConstraint = locale.T("constraint_auto")
			if len(req.VersionConstraints) > 0 {
				reqs := model.MonoModelConstraintsToRequirements(&req.VersionConstraints)
				versionConstraint = model.RequirementsToVersionString(reqs)
			}

			for _, a := range artifacts {
				if a.Namespace == ns.String() && a.Name == req.Requirement {
					versionConstraint = locale.Tr("constraint_resolved", versionConstraint, *a.Version)
					break
				}
			}
		}

		row := packageRow{
			req.Requirement,
			versionConstraint,
		}
		rows = append(rows, row)
	}

	// Sort the rows.
	less := func(i, j int) bool {
		a := rows[i].Pkg
		b := rows[j].Pkg
		if strings.ToLower(a) < strings.ToLower(b) {
			return true
		}
		return a < b
	}
	sort.Slice(rows, less)

	return rows
}
