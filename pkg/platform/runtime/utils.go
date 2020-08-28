package runtime

import (
	"github.com/go-openapi/strfmt"
)

func artifactsToUuids(artifacts []*HeadChefArtifact) []strfmt.UUID {
	result := []strfmt.UUID{}
	for _, v := range artifacts {
		if v.ArtifactID != nil {
			result = append(result, *v.ArtifactID)
		}
	}
	return result
}

func artifactCacheToUuids(artifacts []artifactCacheMeta) []strfmt.UUID {
	result := []strfmt.UUID{}
	for _, v := range artifacts {
		result = append(result, v.ArtifactID)
	}
	return result
}
