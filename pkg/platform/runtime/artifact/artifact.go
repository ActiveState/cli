package artifact

import (
	"fmt"

	"github.com/go-openapi/strfmt"
)

// ArtifactID represents an artifact ID
type ArtifactID = strfmt.UUID

type Named map[ArtifactID]string

// Artifact comprises useful information about an artifact that we extracted from a build plan
type Artifact struct {
	ArtifactID       ArtifactID
	Name             string
	Namespace        string
	Version          *string
	RequestedByOrder bool
	URL              string
	MimeType         string

	GeneratedBy strfmt.UUID

	Dependencies []ArtifactID
}

// Map maps artifact ids to artifact information extracted from a build plan
type Map map[ArtifactID]Artifact

// NamedMap maps artifact names to artifact information extracted from a build plan
type NamedMap map[string]Artifact

// NameWithVersion returns a string <name>@<version> if artifact has a version specified, otherwise it returns just the name
func (a Artifact) NameWithVersion() string {
	version := ""
	if a.Version != nil {
		version = fmt.Sprintf("@%s", *a.Version)
	}
	return a.Name + version
}

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

func ArtifactIDsFromBuildPlanMap(from Map) []ArtifactID {
	ids := make([]ArtifactID, 0, len(from))
	for id := range from {
		ids = append(ids, id)
	}
	return ids
}

func ArtifactIDsFromArtifactSlice(from []Artifact) []ArtifactID {
	ids := make([]ArtifactID, 0, len(from))
	for _, artifact := range from {
		ids = append(ids, artifact.ArtifactID)
	}
	return ids
}
