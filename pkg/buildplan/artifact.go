package buildplan

import "github.com/go-openapi/strfmt"

// Artifact represents a downloadable artifact.
// This artifact may or may not be installable by the State Tool.
type Artifact struct {
	Type                string        `json:"__typename"`
	NodeID              strfmt.UUID   `json:"nodeId"`
	DisplayName         string        `json:"displayName"`
	MimeType            string        `json:"mimeType"`
	GeneratedBy         strfmt.UUID   `json:"generatedBy"`
	RuntimeDependencies []strfmt.UUID `json:"runtimeDependencies"`
	Status              string        `json:"status"`
	URL                 string        `json:"url"`
	LogURL              string        `json:"logURL"`
	Checksum            string        `json:"checksum"`

	IsRuntimeDependency   bool
	IsBuildtimeDependency bool

	Platform    *strfmt.UUID
	Ingredients []*Ingredient // While most artifacts only have a single ingredient, some artifacts such as installers can have multiple.

	terminal       string    // We don't want to expose this, because terminals are a low level concept
	parentArtifact *Artifact // We don't want to expose this, because the concept of a parent artifact is unique to low level buildplan logic
}

type Artifacts []*Artifact

func (a Artifacts) ToIDMap() map[strfmt.UUID]*Artifact {
	result := make(map[strfmt.UUID]*Artifact, len(a))
	for _, a := range a {
		result[a.NodeID] = a
	}
	return result
}

func (a Artifacts) ToNameMap() map[string]*Artifact {
	result := make(map[string]*Artifact, len(a))
	for _, a := range a {
		name := a.DisplayName
		if len(a.Ingredients) == 0 {
			name = a.Ingredients[0].Name
		}
		result[name] = a
	}
	return result
}

type ArtifactChangeset struct {
	Added   []*Artifact
	Removed []*Artifact
	Updated []ArtifactUpdate
}

type ArtifactUpdate struct {
	From *Artifact
	To   *Artifact
}
