package runtime_test

import (
	"path"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/pkg/platform/runtime"
)

func headchefArtifact(artifactPath string) (*runtime.HeadChefArtifact, map[string]*runtime.HeadChefArtifact) {
	artifactID := strfmt.UUID("00010001-0001-0001-0001-000100010001")
	uri := strfmt.URI("https://test.tld/" + path.Join(artifactPath))
	artifact := &runtime.HeadChefArtifact{
		ArtifactID: &artifactID,
		URI:        uri,
	}
	archives := map[string]*runtime.HeadChefArtifact{}
	archives[artifactPath] = artifact
	return artifact, archives
}
