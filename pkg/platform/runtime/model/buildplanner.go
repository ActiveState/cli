package model

import (
	"net/http"
	"net/url"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/gqlclient"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/model"
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
		client: gqlclient.NewWithOpts("https://platform-internal.activestate.com/sv/buildplanner/graphql", 0, graphql.WithHTTPClient(&http.Client{})),
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

	if model.BuildPlanStatusEnum(resp.Execute.Status) != model.Ready {
		return nil, locale.NewError("err_buildplanner_not_ready", "Build plan is not ready")
	}

	// logging.Debug("Number of artifacts: %d", len(resp.Execute.Artifacts))
	// logging.Debug("Number of steps: %d", len(resp.Execute.Steps))

	// targetIDs are the IDs of the artifacts that the user requested
	var targetIDs []string
	for _, terminal := range resp.Terminals {
		// TODO: This should filter by platform in the tag.
		// Should the build graph even return this if it's not for a platform we're on?
		// Can we filter this out in the results in our build query?
		// Using the tag to filter this out is esentially string matching, which we should avoid
		targetIDs = append(targetIDs, terminal.TargetIDs...)
	}

	var runtimeDeps []model.Artifact
	var targetArtifacts []model.Artifact
	for _, tID := range targetIDs {
		for _, artifact := range resp.Artifacts {
			if artifact.TargetID == tID {
				targetArtifacts = append(targetArtifacts, artifact)
				runtimeDeps = runtimeDependencies(artifact.TargetID, resp.Artifacts)
			}
		}
	}

	var seen []string
	var uniqueDeps []model.Artifact
	for _, dep := range runtimeDeps {
		if !funk.Contains(seen, dep.TargetID) {
			seen = append(seen, dep.TargetID)
			uniqueDeps = append(uniqueDeps, dep)
		}
	}
	runtimeDeps = uniqueDeps

	// logging.Debug("Target artifacts: %v", targetArtifacts)
	// logging.Debug("Runtime dependencies: %v", runtimeDeps)

	names := make(map[string]model.Artifact)

	for _, t := range targetArtifacts {
		name, err := getArtifactName(t.GeneratedBy, resp.Steps, resp.Sources)
		if err != nil {
			return nil, locale.WrapError(err, "err_get_artifact_name", "Could not get artifact name")
		}
		names[name] = t
	}

	for _, d := range runtimeDeps {
		name, err := getArtifactName(d.GeneratedBy, resp.Steps, resp.Sources)
		if err != nil {
			return nil, locale.WrapError(err, "err_get_artifact_name", "Could not get artifact name")
		}
		names[name] = d
	}

	// logging.Debug("Names: %+v", names)
	// logging.Debug("response: %#v", resp)

	return &BuildResult{
		BuildEngine: Alternative,
		BuildPlan:   resp,
		BuildReady:  model.BuildPlanStatusEnum(resp.Status) == model.Ready,
	}, nil
}

// var depth int

func runtimeDependencies(baseID string, artifacts []model.Artifact) []model.Artifact {
	// logging.Debug("Depth: %d", depth)
	// depth++

	var deps []model.Artifact
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
