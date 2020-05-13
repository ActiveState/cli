package runtime_test

import (
	"fmt"
	"path"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/pkg/platform/runtime"
)

func headchefArtifact(artifactPath string) (*runtime.HeadChefArtifact, map[string]*runtime.HeadChefArtifact) {
	artifactID := strfmt.UUID("00010001-0001-0001-0001-000100010001")
	ingredientVersionID := strfmt.UUID("00030003-0003-0003-0003-000300030003")
	uri := strfmt.URI("https://test.tld/" + path.Join(artifactPath))
	artifact := &runtime.HeadChefArtifact{
		ArtifactID:          &artifactID,
		IngredientVersionID: ingredientVersionID,
		URI:                 uri,
	}
	archives := map[string]*runtime.HeadChefArtifact{}
	archives[artifactPath] = artifact
	return artifact, archives
}

type artifactsResultMockOption func(*runtime.FetchArtifactsResult) *runtime.FetchArtifactsResult

func mockFetchArtifactsResult(options ...artifactsResultMockOption) *runtime.FetchArtifactsResult {
	recipeID := strfmt.UUID("00020002-0002-0002-0002-0002-00020000200002")
	res := &runtime.FetchArtifactsResult{
		RecipeID:    recipeID,
		Artifacts:   []*runtime.HeadChefArtifact{},
		BuildEngine: runtime.Alternative,
	}
	for _, opt := range options {
		res = opt(res)
	}
	return res
}

func withRegularArtifacts(numArtifacts int) artifactsResultMockOption {
	return func(res *runtime.FetchArtifactsResult) *runtime.FetchArtifactsResult {

		for i := 0; i < numArtifacts; i++ {
			uri := fmt.Sprintf("https://test.tld/artifact%d/artifact.tar.gz", i)
			artifactID := strfmt.UUID(fmt.Sprintf("00010001-0001-0001-0001-00010001000%d", i))
			ingredientVersionID := strfmt.UUID(fmt.Sprintf("00020001-0001-0001-0001-00010001000%d", i))
			res.Artifacts = append(res.Artifacts, &runtime.HeadChefArtifact{
				ArtifactID:          &artifactID,
				IngredientVersionID: ingredientVersionID,
				URI:                 strfmt.URI(uri),
			})
		}
		return res
	}
}

func withURIArtifact(uri string) artifactsResultMockOption {
	return func(res *runtime.FetchArtifactsResult) *runtime.FetchArtifactsResult {

		artifactID := strfmt.UUID("00010003-0001-0001-0001-000100010001")
		ingredientVersionID := strfmt.UUID("00020001-0001-0001-0001-000100010001")
		res.Artifacts = append(res.Artifacts, &runtime.HeadChefArtifact{
			ArtifactID:          &artifactID,
			IngredientVersionID: ingredientVersionID,
			URI:                 strfmt.URI(uri),
		})
		return res
	}
}

func withTerminalArtifacts(numArtifacts int) artifactsResultMockOption {
	return func(res *runtime.FetchArtifactsResult) *runtime.FetchArtifactsResult {

		for i := 0; i < numArtifacts; i++ {
			uri := fmt.Sprintf("https://test.tld/terminal_artifact%d/artifact.tar.gz", i)
			artifactID := strfmt.UUID(fmt.Sprintf("00010002-0001-0001-0001-00010001000%d", i))
			res.Artifacts = append(res.Artifacts, &runtime.HeadChefArtifact{
				ArtifactID: &artifactID,
				URI:        strfmt.URI(uri),
			})
		}
		return res
	}
}

// camelInstallerExtension returns the installer extension for camel artifacts without caring
// about the CamelRuntime.  This is useful to mock OS independent artifact file names.
func camelInstallerExtension() string {
	crTemp := &runtime.CamelRuntime{}
	return crTemp.InstallerExtension()
}
