package packages

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/ActiveState/cli/pkg/buildplan"
	bpModel "github.com/ActiveState/cli/pkg/platform/model/buildplanner"
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/pkg/localcommit"
	gqlModel "github.com/ActiveState/cli/pkg/platform/api/graphql/model"
	"github.com/ActiveState/cli/pkg/platform/authentication"
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
	out       output.Outputer
	project   *project.Project
	analytics analytics.Dispatcher
	svcModel  *model.SvcModel
	auth      *authentication.Auth
	cfg       *config.Instance
}

// NewList prepares a list execution context for use.
func NewList(prime primeable) *List {
	return &List{
		out:       prime.Output(),
		project:   prime.Project(),
		analytics: prime.Analytics(),
		svcModel:  prime.SvcModel(),
		auth:      prime.Auth(),
		cfg:       prime.Config(),
	}
}

type requirement struct {
	Name            string `json:"package"`
	Version         string `json:"version" `
	ResolvedVersion string `json:"resolved_version"`
}

type requirementPlainOutput struct {
	Name    string `locale:"package_name,Name"`
	Version string `locale:"package_version,Version"`
}

// Run executes the list behavior.
func (l *List) Run(params ListRunParams, nstype model.NamespaceType) error {
	logging.Debug("ExecuteList")

	l.out.Notice(locale.T("manifest_deprecation_warning"))

	if l.project != nil && params.Project == "" {
		l.out.Notice(locale.Tr("operating_message", l.project.NamespaceString(), l.project.Dir()))
	}

	var commitID *strfmt.UUID
	var err error
	switch {
	case params.Commit != "":
		commitID, err = targetFromCommit(params.Commit, l.project)
		if err != nil {
			return locale.WrapError(err, fmt.Sprintf("%s_err_cannot_obtain_commit", nstype))
		}
	case params.Project != "":
		commitID, err = targetFromProject(params.Project)
		if err != nil {
			return locale.WrapError(err, fmt.Sprintf("%s_err_cannot_obtain_commit", nstype))
		}
	default:
		commitID, err = targetFromProjectFile(l.project)
		if err != nil {
			return locale.WrapError(err, fmt.Sprintf("%s_err_cannot_obtain_commit", nstype))
		}
	}

	checkpoint, err := fetchCheckpoint(commitID, l.auth)
	if err != nil {
		return locale.WrapError(err, fmt.Sprintf("%s_err_cannot_fetch_checkpoint", nstype))
	}

	language, err := model.LanguageByCommit(*commitID, l.auth)
	if err != nil {
		return locale.WrapError(err, "err_package_list_language", "Unable to get language from project")
	}
	var ns *model.Namespace
	if language.Name != "" {
		ns = ptr.To(model.NewNamespacePkgOrBundle(language.Name, nstype))
	}

	// Fetch resolved artifacts list for showing full version numbers, if possible.
	var artifacts buildplan.Artifacts
	if l.project != nil && params.Project == "" {
		bpm := bpModel.NewBuildPlannerModel(l.auth, l.svcModel)
		commit, err := bpm.FetchCommit(*commitID, l.project.Owner(), l.project.Name(), l.project.BranchName(), nil)
		if err != nil {
			return errs.Wrap(err, "could not fetch commit")
		}
		artifacts = commit.BuildPlan().Artifacts(buildplan.FilterStateArtifacts())
	}

	requirements := model.FilterCheckpointNamespace(checkpoint, model.NamespacePackage, model.NamespaceBundle)
	sort.SliceStable(requirements, func(i, j int) bool {
		return strings.ToLower(requirements[i].Requirement) < strings.ToLower(requirements[j].Requirement)
	})

	requirementsPlainOutput := []requirementPlainOutput{}
	requirementsOutput := []requirement{}

	for _, req := range requirements {
		if !strings.Contains(strings.ToLower(req.Requirement), strings.ToLower(params.Name)) {
			continue
		}

		if !strings.HasPrefix(req.Namespace, nstype.Prefix()) {
			continue
		}

		version := req.VersionConstraint
		if version == "" {
			version = model.GqlReqVersionConstraintsString(req)
			if version == "" {
				version = locale.T("constraint_auto")
			}
		}

		resolvedVersion := ""
		(func() {
			for _, a := range artifacts {
				for _, i := range a.Ingredients {
					if ns != nil && i.Namespace == ns.String() && i.Name == req.Requirement {
						resolvedVersion = i.Version
						return // break outer loop
					}
				}
			}
		})()

		plainVersion := version
		if resolvedVersion != "" && resolvedVersion != version {
			plainVersion = locale.Tr("constraint_resolved", version, resolvedVersion)
		}
		requirementsPlainOutput = append(requirementsPlainOutput, requirementPlainOutput{
			Name:    req.Requirement,
			Version: plainVersion,
		})

		requirementsOutput = append(requirementsOutput, requirement{
			Name:            req.Requirement,
			Version:         version,
			ResolvedVersion: resolvedVersion,
		})
	}

	var plainOutput interface{} = requirementsPlainOutput
	if len(requirementsOutput) == 0 {
		plainOutput = locale.T(fmt.Sprintf("%s_list_no_packages", nstype.String()))
	}

	l.out.Print(output.Prepare(plainOutput, requirementsOutput))
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
		return nil, rationalize.ErrNoProject
	}
	commit, err := localcommit.Get(proj.Dir())
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

func fetchCheckpoint(commit *strfmt.UUID, auth *authentication.Auth) ([]*gqlModel.Requirement, error) {
	if commit == nil {
		logging.Debug("commit id is nil")
		return nil, nil
	}

	checkpoint, _, err := model.FetchCheckpointForCommit(*commit, auth)
	if err != nil && errors.Is(err, model.ErrNoData) {
		return nil, locale.WrapExternalError(err, "package_no_data")
	}

	return checkpoint, err
}
