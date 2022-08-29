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
	"github.com/thoas/go-funk"
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

	if model.BuildPlanStatus(resp.Project.Commit.Build.Status) != model.BuildReady {
		return nil, locale.NewError("err_buildplanner_not_ready", "Build plan is not ready")
	}

	originalTargets := resp.Project.Commit.Build.Targets
	var targets []model.Target
	for _, terminal := range resp.Project.Commit.Build.Terminals {
		// TODO: Add proper tag handling
		if terminal.Tag == "orphans" {
			continue
		}
		for _, id := range terminal.TargetIDs {
			targets = append(targets, runtimeDependencies(id, resp.Project.Commit.Build.Targets)...)
		}
	}
	var uniqueTargets []model.Target
	for _, target := range targets {
		if !funk.Contains(uniqueTargets, target) {
			uniqueTargets = append(uniqueTargets, target)
		}
	}
	resp.Project.Commit.Build.Targets = uniqueTargets

	return &BuildResult{
		BuildEngine: Alternative,
		BuildPlan:   processOriginalTargets(resp, originalTargets),
		BuildReady:  model.BuildPlanStatus(resp.Project.Commit.Build.Status) == model.BuildReady,
	}, nil
}

// TODO: Tempoarary function, remove after dependecy resolution is updated
func processOriginalTargets(bp *model.BuildPlan, orignal []model.Target) *model.BuildPlan {
	var steps []model.Step
	var sources []model.Source
	for _, artifact := range orignal {
		if artifact.Type == "Step" {
			steps = append(steps, model.Step{
				TargetID: artifact.TargetID,
				Inputs:   artifact.Inputs,
				Outputs:  artifact.Outputs,
			})
			continue
		}
		if artifact.Type == "Source" {
			sources = append(sources, model.Source{
				TargetID:  artifact.TargetID,
				Name:      artifact.Name,
				Namespace: artifact.Namespace,
				Version:   artifact.Version,
			})
			continue
		}
	}
	bp.Project.Commit.Build.Steps = steps
	bp.Project.Commit.Build.Sources = sources
	return bp
}

func runtimeDependencies(baseID string, artifacts []model.Target) []model.Target {
	var deps []model.Target
	for _, artifact := range artifacts {
		if artifact.TargetID == baseID {
			for _, id := range artifact.RuntimeDependencies {
				deps = append(deps, artifact)
				deps = append(deps, runtimeDependencies(id, artifacts)...)
			}
		}
	}
	return deps
}

func getArtifactName(generatedByID string, steps []model.Step, sources []model.Source) (string, error) {
	for _, step := range steps {
		if step.TargetID != generatedByID {
			continue
		}

		for _, input := range step.Inputs {
			if input.Tag == "src" {
				// Should only be one source per step
				for _, id := range input.TargetIDs {
					for _, src := range sources {
						if src.TargetID == id {
							return src.Name, nil
						}
					}
				}
			}
		}
	}
	return "", locale.NewError("err_resolve_artifact_name", "Could not resolve artifact name")
}
