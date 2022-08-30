package model

import (
	"net/http"
	"net/url"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/gqlclient"
	"github.com/ActiveState/cli/internal/locale"
	model "github.com/ActiveState/cli/pkg/platform/api/graphql/model/buildplan"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/request"
	"github.com/ActiveState/cli/pkg/platform/api/headchef"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
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

func (b *BuildPlanner) ResolveRecipe(commitID strfmt.UUID, owner, projectName string) (*inventory_models.Recipe, error) {
	return nil, errs.New("not implemented")
}

func (b *BuildPlanner) RequestBuild(recipeID, commitID strfmt.UUID, owner, project string) (headchef.BuildStatusEnum, *headchef_models.V1BuildStatusResponse, error) {
	return headchef.Error, nil, errs.New("not implemented")
}

func (b *BuildPlanner) SignS3URL(uri *url.URL) (*url.URL, error) {
	return nil, errs.New("not implemented")
}

func (b *BuildPlanner) FetchBuildResult(commitID strfmt.UUID, _, _ string) (*BuildResult, error) {
	resp := &model.BuildPlan{}
	err := b.client.Run(request.BuildPlanByCommitID(commitID.String()), resp)
	if err != nil {
		return nil, errs.Wrap(err, "failed to fetch build plan")
	}

	// This is a lot of awkward error checking
	if resp.Project.Commit.Build.Status != model.BuildReady {
		return nil, locale.NewError("err_buildplanner_not_ready", "Build plan is not ready")
	}
	if resp.Project.Type == model.ProjectNotFoundType {
		return nil, locale.NewError("err_buildplanner_project_not_found", "Project not found")
	}
	if resp.Project.Commit.Type == model.CommitNotFoundType {
		return nil, locale.NewError("err_buildplanner_commit_not_found", "Commit not found")
	}
	if resp.Project.Commit.Build.Type == model.BuildResultPlanningError {
		// Need a failed build from the test harness to properly handle errors
		// TODO: Further unwrap errors from the build planner
		return nil, locale.NewError("err_buildplanner_build_error", "Build encountered an error: {{.V0}}", resp.Project.Commit.Build.Error)
	}

	return &BuildResult{
		BuildEngine: Alternative,
		Build:       &resp.Project.Commit.Build,
		BuildReady:  model.BuildPlanStatus(resp.Project.Commit.Build.Status) == model.BuildReady,
	}, nil
}
