package model

import (
	"net/url"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/api/headchef"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
)

// var _ runtime.ClientProvider = &Default{}

// Model is the default client that actually talks to the backend
type Model struct{}

// NewDefault is the constructor for the Model client
func NewDefault() *Model {
	return &Model{}
}

func (m *Model) ResolveRecipe(commitID strfmt.UUID, owner, projectName string) (*inventory_models.Recipe, error) {
	return model.ResolveRecipe(commitID, owner, projectName)
}

func (m *Model) RequestBuild(recipeID, commitID strfmt.UUID, owner, project string) (headchef.BuildStatusEnum, *headchef_models.BuildStatusResponse, error) {
	return model.RequestBuild(recipeID, commitID, owner, project)
}

func (m *Model) SignS3URL(uri *url.URL) (*url.URL, error) {
	return model.SignS3URL(uri)
}

// BuildResult is the unified response of a Build request
type BuildResult struct {
	BuildEngine         BuildEngine
	Recipe              *inventory_models.Recipe
	BuildStatusResponse *headchef_models.BuildStatusResponse
	BuildStatus         headchef.BuildStatusEnum
	BuildReady          bool
}

func (b *BuildResult) OrderedArtifacts() []artifact.ArtifactID {
	res := make([]artifact.ArtifactID, 0, len(b.BuildStatusResponse.Artifacts))
	for _, a := range b.BuildStatusResponse.Artifacts {
		res = append(res, *a.ArtifactID)
	}
	return res
}

// FetchBuildResult requests a build for a resolved recipe and returns the result in a BuildResult struct
func (m *Model) FetchBuildResult(commitID strfmt.UUID, owner, project string) (*BuildResult, error) {
	recipe, err := m.ResolveRecipe(commitID, owner, project)
	if err != nil {
		return nil, locale.WrapError(err, "setup_build_resolve_recipe_err", "Could not resolve recipe for project {{.V0}}/{{.V1}}#{{.V2}}", owner, project, commitID.String())
	}

	bse, resp, err := m.RequestBuild(*recipe.RecipeID, commitID, owner, project)
	if err != nil {
		return nil, locale.WrapError(err, "headchef_build_err", "Could not request build for {{.V0}}/{{.V1}}#{{.V2}}", owner, project, commitID.String())
	}

	engine := buildEngineFromResponse(resp)

	return &BuildResult{
		BuildEngine:         engine,
		Recipe:              recipe,
		BuildStatusResponse: resp,
		BuildStatus:         bse,
		BuildReady:          bse == headchef.Completed,
	}, nil
}
