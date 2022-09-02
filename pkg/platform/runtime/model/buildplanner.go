package model

import (
	"net/http"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/gqlclient"
	"github.com/ActiveState/cli/internal/locale"
	model "github.com/ActiveState/cli/pkg/platform/api/graphql/model/buildplan"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/request"
	"github.com/go-openapi/strfmt"
	"github.com/machinebox/graphql"
)

type BuildPlanner struct {
	client *gqlclient.Client
}

func NewBuildPlanner() *BuildPlanner {
	return &BuildPlanner{
		client: gqlclient.NewWithOpts("https://platform.activestate.com/sv/buildplanner/graphql", 0, graphql.WithHTTPClient(&http.Client{})),
	}
}

func (bp *BuildPlanner) FetchBuildResult(commitID strfmt.UUID, _, _ string) (*BuildResult, error) {
	resp := &model.BuildPlan{}
	err := bp.client.Run(request.BuildPlanByCommitID(commitID.String()), resp)
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
		// Need a failed build from the test harness to properly handle errors
		// TODO: Further unwrap errors from the build planner
		return nil, locale.NewError("err_buildplanner_build_error", "Build encountered an error: {{.V0}}", resp.Project.Commit.Build.Error)
	}

	// The type aliasing in the query populates the
	// response with emtpy targets that we have to remove
	removeEmptyTargets(resp)

	return &BuildResult{
		BuildEngine: Alternative,
		Build:       resp.Project.Commit.Build,
		BuildReady:  resp.Project.Commit.Build.Status == model.BuildReady,
	}, nil
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
