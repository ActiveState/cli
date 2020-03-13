package runtime_test

import (
	"path"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/pkg/platform/runtime"
)

func headchefArtifact(artifactPath string) map[string]*runtime.HeadChefArtifact {
	artifactID := strfmt.UUID("00010001-0001-0001-0001-000100010001")
	uri := strfmt.URI("https://test.tld/" + path.Join(artifactPath))
	result := map[string]*runtime.HeadChefArtifact{}
	result[artifactPath] = &runtime.HeadChefArtifact{
		ArtifactID: &artifactID,
		URI:        uri,
	}
	return result
}
