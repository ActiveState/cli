package model

import (
	"net/http"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/gqlclient"
	"github.com/ActiveState/cli/internal/locale"
	model "github.com/ActiveState/cli/pkg/platform/api/graphql/model/buildplan"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/request"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/go-openapi/strfmt"
	"github.com/machinebox/graphql"
)

type BuildPlannerError struct {
	wrapped          error
	validationErrors []string
	isTransient      bool
}

func (e *BuildPlannerError) Error() string {
	return "buildplan_error"
}

func (e *BuildPlannerError) Unwrap() error {
	return e.wrapped
}

func (e *BuildPlannerError) ValidationErrors() []string {
	return e.validationErrors
}

func (e *BuildPlannerError) IsTransient() bool {
	return e.isTransient
}

type BuildPlanner struct {
	auth   *authentication.Auth
	client *gqlclient.Client
	def    *Model
}

func NewBuildPlanner(auth *authentication.Auth) *BuildPlanner {
	return &BuildPlanner{
		auth:   auth,
		client: gqlclient.NewWithOpts("https://platform.activestate.com/sv/buildplanner/graphql", 0, graphql.WithHTTPClient(&http.Client{})),
		def:    NewDefault(auth),
	}
}

func (bp *BuildPlanner) FetchBuildResult(commitID strfmt.UUID, owner, project string) (*BuildResult, error) {
	resp := &model.BuildPlan{}
	err := bp.client.Run(request.BuildPlan(owner, project, commitID.String()), resp)
	if err != nil {
		return nil, errs.Wrap(err, "failed to fetch build plan")
	}

	// This is a lot of awkward error checking
	if resp.Project.Type == model.ProjectNotFoundType {
		return nil, locale.NewError("err_buildplanner_project_not_found", "Build plan does not contain project")
	}
	if resp.Project.Commit.Type == model.CommitNotFoundType {
		return nil, locale.NewError("err_buildplanner_commit_not_found", "Build plan does not contain commit")
	}
	if resp.Project.Commit.Build.Type == model.BuildResultPlanningError {
		var errs []string
		var isTransient bool
		for _, se := range resp.Project.Commit.Build.SubErrors {
			errs = append(errs, se.Message)
			isTransient = se.IsTransient
		}
		return nil, &BuildPlannerError{
			wrapped:          locale.NewError("err_buildplanner", resp.Project.Commit.Build.Error),
			validationErrors: errs,
			isTransient:      isTransient,
		}
	}

	// The type aliasing in the query populates the
	// response with emtpy targets that we have to remove
	removeEmptyTargets(resp)

	res := BuildResult{
		BuildEngine: Alternative,
		Build:       resp.Project.Commit.Build,
		BuildReady:  resp.Project.Commit.Build.Status == model.BuildReady,
	}

	if resp.Project.Commit.Build.Status == model.BuildBuilding {
		res.Recipe, err = bp.def.ResolveRecipe(commitID, owner, project)
		if err != nil {
			return nil, locale.WrapError(err, "setup_build_resolve_recipe_err", "Could not resolve recipe for project {{.V0}}/{{.V1}}#{{.V2}}", owner, project, commitID.String())
		}
	}

	return &res, nil
}

func removeEmptyTargets(bp *model.BuildPlan) {
	var steps []*model.Step
	for _, step := range bp.Project.Commit.Build.Steps {
		if step.TargetID == "" {
			continue
		}
		steps = append(steps, step)
	}

	var sources []*model.Source
	for _, source := range bp.Project.Commit.Build.Sources {
		if source.TargetID == "" {
			continue
		}
		sources = append(sources, source)
	}

	var artifacts []*model.Artifact
	for _, artifact := range bp.Project.Commit.Build.Artifacts {
		if artifact.TargetID == "" {
			continue
		}
		artifacts = append(artifacts, artifact)
	}

	bp.Project.Commit.Build.Steps = steps
	bp.Project.Commit.Build.Sources = sources
	bp.Project.Commit.Build.Artifacts = artifacts
}
