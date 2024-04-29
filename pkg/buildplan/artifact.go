package buildplan

import (
	"reflect"
	"sort"

	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/sliceutils"
	"github.com/ActiveState/cli/pkg/buildplan/raw"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	"github.com/go-openapi/strfmt"
)

// Artifact represents a downloadable artifact.
// This artifact may or may not be installable by the State Tool.
type Artifact struct {
	raw *raw.Artifact // Don't expose as this may lead to external packages using low level buildplan logic

	ArtifactID  strfmt.UUID
	DisplayName string
	MimeType    string
	URL         string
	LogURL      string
	Errors      []string
	Checksum    string
	Status      string

	IsRuntimeDependency   bool
	IsBuildtimeDependency bool

	Platforms   []strfmt.UUID
	Ingredients []*Ingredient `json:"-"` // While most artifacts only have a single ingredient, some artifacts such as installers can have multiple.

	children []*Artifact
}

// Name returns the name of the ingredient for this artifact, if it only has exactly one ingredient associated.
// Otherwise it returns the DisplayName, which is less reliable and consistent.
func (a *Artifact) Name() string {
	if len(a.Ingredients) == 1 {
		return a.Ingredients[0].Name
	}
	logging.Debug("Using displayname because artifact has %d ingredients", len(a.Ingredients))
	return a.DisplayName
}

// Version returns the name of the ingredient for this artifact, if it only has exactly one ingredient associated.
// Otherwise it returns an empty version.
func (a *Artifact) Version() string {
	if len(a.Ingredients) == 1 {
		return a.Ingredients[0].Version
	}
	return ""
}

func (a *Artifact) NameAndVersion() string {
	version := a.Version()
	if version == "" {
		return a.Name()
	}
	return a.Name() + "@" + version
}

type Artifacts []*Artifact

type ArtifactIDMap map[strfmt.UUID]*Artifact

type ArtifactNameMap map[string]*Artifact

func (a Artifacts) Filter(filters ...FilterArtifact) Artifacts {
	if len(filters) == 0 {
		return a
	}
	artifacts := []*Artifact{}
	for _, ar := range a {
		include := true
		for _, filter := range filters {
			if !filter(ar) {
				include = false
				break
			}
		}
		if include {
			artifacts = append(artifacts, ar)
		}
	}
	return artifacts
}

func (a Artifacts) Ingredients() Ingredients {
	result := Ingredients{}
	for _, a := range a {
		result = append(result, a.Ingredients...)
	}
	return sliceutils.Unique(result)
}

func (a Artifacts) ToIDMap() ArtifactIDMap {
	result := make(map[strfmt.UUID]*Artifact, len(a))
	for _, a := range a {
		result[a.ArtifactID] = a
	}
	return result
}

func (a Artifacts) ToIDSlice() []strfmt.UUID {
	result := make([]strfmt.UUID, len(a))
	for n, a := range a {
		result[n] = a.ArtifactID
	}
	return result
}

func (a Artifacts) ToNameMap() ArtifactNameMap {
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

func (a ArtifactUpdate) VersionsChanged() bool {
	fromVersions := []string{}
	for _, ing := range a.From.Ingredients {
		fromVersions = append(fromVersions, ing.Version)
	}
	sort.Strings(fromVersions)
	toVersions := []string{}
	for _, ing := range a.To.Ingredients {
		toVersions = append(toVersions, ing.Version)
	}
	sort.Strings(toVersions)

	return !reflect.DeepEqual(fromVersions, toVersions)
}

func (a *Artifact) Dependencies(recursive bool) Artifacts {
	dependencies := a.children
	if recursive {
		for _, ac := range a.children {
			dependencies = append(dependencies, ac.Dependencies(recursive)...)
		}
	}
	return dependencies
}

// SetDownload is used to update the URL and checksum of an artifact. This allows us to keep using the same artifact
// type, while also facilitating dressing up in-progress artifacts with their download info later on
func (a *Artifact) SetDownload(uri string, checksum string) {
	a.URL = uri
	a.Checksum = checksum
	a.Status = types.ArtifactSucceeded
}
