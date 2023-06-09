package artifact

import (
	"github.com/go-openapi/strfmt"
)

// ArtifactID represents an artifact ID
type ArtifactID = strfmt.UUID

type Named map[ArtifactID]string

func ResolveArtifactNames(resolver func(ArtifactID) string, artifacts []ArtifactID) Named {
	names := Named{}
	for _, id := range artifacts {
		names[id] = resolver(id)
	}
	return names
}

func ArtifactIDsToMap(ids []ArtifactID) map[ArtifactID]struct{} {
	idmap := make(map[ArtifactID]struct{})
	for _, id := range ids {
		idmap[id] = struct{}{}
	}
	return idmap
}

func ArtifactIDsFromBuildPlanMap(from ArtifactBuildPlanMap) []ArtifactID {
	ids := make([]ArtifactID, 0, len(from))
	for id := range from {
		ids = append(ids, id)
	}
	return ids
}
