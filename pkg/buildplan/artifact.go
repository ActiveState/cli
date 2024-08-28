package buildplan

import (
	"reflect"
	"sort"

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

	Ingredients []*Ingredient `json:"-"` // While most artifacts only have a single ingredient, some artifacts such as installers can have multiple.

	isRuntimeDependency   bool
	isBuildtimeDependency bool

	platforms []strfmt.UUID
	children  []ArtifactRelation
}

type Relation int

const (
	RuntimeRelation Relation = iota
	BuildtimeRelation
)

type ArtifactRelation struct {
	Artifact *Artifact
	Relation Relation
}

// Name returns the name of the ingredient for this artifact, if it only has exactly one ingredient associated.
// Otherwise it returns the DisplayName, which is less reliable and consistent.
func (a *Artifact) Name() string {
	if len(a.Ingredients) == 1 {
		return a.Ingredients[0].Name
	}
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

type ChangeType int

const (
	ArtifactAdded ChangeType = iota
	ArtifactRemoved
	ArtifactUpdated
)

func (c ChangeType) String() string {
	switch c {
	case ArtifactAdded:
		return "added"
	case ArtifactRemoved:
		return "removed"
	case ArtifactUpdated:
		return "updated"
	}

	return "unknown"
}

type ArtifactChange struct {
	ChangeType ChangeType
	Artifact   *Artifact
	Old        *Artifact // Old is only set when ChangeType=ArtifactUpdated
}

type ArtifactChangeset []ArtifactChange

func (a ArtifactChangeset) Filter(t ...ChangeType) ArtifactChangeset {
	lookup := make(map[ChangeType]struct{}, len(t))
	for _, t := range t {
		lookup[t] = struct{}{}
	}
	result := ArtifactChangeset{}
	for _, ac := range a {
		if _, ok := lookup[ac.ChangeType]; ok {
			result = append(result, ac)
		}
	}
	return result
}

func (a ArtifactChange) VersionsChanged() bool {
	if a.Old == nil {
		return false
	}
	fromVersions := []string{}
	for _, ing := range a.Old.Ingredients {
		fromVersions = append(fromVersions, ing.Version)
	}
	sort.Strings(fromVersions)
	toVersions := []string{}
	for _, ing := range a.Artifact.Ingredients {
		toVersions = append(toVersions, ing.Version)
	}
	sort.Strings(toVersions)

	return !reflect.DeepEqual(fromVersions, toVersions)
}

func (as Artifacts) RuntimeDependencies(recursive bool, ignore *map[strfmt.UUID]struct{}) Artifacts {
	dependencies := Artifacts{}
	for _, a := range as {
		dependencies = append(dependencies, a.dependencies(recursive, ignore, RuntimeRelation)...)
	}
	return dependencies
}

func (a *Artifact) RuntimeDependencies(recursive bool, ignore *map[strfmt.UUID]struct{}) Artifacts {
	return a.dependencies(recursive, ignore, RuntimeRelation)
}

// Dependencies returns ALL dependencies that an artifact has, this covers runtime and build time dependencies.
// It does not cover test dependencies as we have no use for them in the state tool.
func (as Artifacts) Dependencies(recursive bool, ignore *map[strfmt.UUID]struct{}) Artifacts {
	dependencies := Artifacts{}
	for _, a := range as {
		dependencies = append(dependencies, a.dependencies(recursive, ignore, RuntimeRelation, BuildtimeRelation)...)
	}
	return dependencies
}

// Dependencies returns ALL dependencies that an artifact has, this covers runtime and build time dependencies.
// It does not cover test dependencies as we have no use for them in the state tool.
func (a *Artifact) Dependencies(recursive bool, ignore *map[strfmt.UUID]struct{}) Artifacts {
	return a.dependencies(recursive, ignore, RuntimeRelation, BuildtimeRelation)
}

func (a *Artifact) dependencies(recursive bool, maybeIgnore *map[strfmt.UUID]struct{}, relations ...Relation) Artifacts {
	ignore := map[strfmt.UUID]struct{}{}
	if maybeIgnore != nil {
		ignore = *maybeIgnore
	}

	// Guard against recursion, this shouldn't really be possible but we don't know how the buildplan might evolve
	// so better safe than sorry.
	if _, ok := ignore[a.ArtifactID]; ok {
		return Artifacts{}
	}
	ignore[a.ArtifactID] = struct{}{}

	dependencies := Artifacts{}
	for _, ac := range a.children {
		related := len(relations) == 0
		for _, relation := range relations {
			if ac.Relation == relation {
				related = true
			}
		}
		if !related {
			continue
		}

		if _, ok := ignore[ac.Artifact.ArtifactID]; !ok {
			dependencies = append(dependencies, ac.Artifact)
			if recursive {
				dependencies = append(dependencies, ac.Artifact.dependencies(recursive, &ignore, relations...)...)
			}
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
