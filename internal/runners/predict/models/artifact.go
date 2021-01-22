package models

import (
	"fmt"
	"time"

	uuid "github.com/satori/go.uuid"
)

// Artifact represents an individual step of the build. At the most abstract
// level, a build step takes some kind of input (source code and/or dependency
// artifacts), processes the input, and produces output which is packaged up as
// a new artifact. This can be building source code into an executable or
// packaging up already-built executables into some kind of other distributable
// like an installer.
type Artifact struct {
	// ArtifactID is a unique but consistently-generated UUID for the artifact.
	ArtifactID uuid.UUID
	// BuildState is the stage of the building process that this artifact is in.
	BuildState BuildState
	// ReadyAt is the earliest time at which the next attempt to build this
	// artifact should be made. Only set if BuildState is READY.
	ReadyAt *time.Time
	// PreviousAttempts is the number of previously executed attempts to build
	// this artifact that have failed.
	PreviousAttempts uint
	// URI is the URI in S3 at which this artifact is stored. Only set if
	// BuildState is SUCCEEDED.
	URI *string
	// Checksum of the artifact's archive file (stored at URI). Only set if
	// BuildState is SUCCEEDED.
	Checksum *string
	// ErrorMessage detailing in a human-readable form why the artifact failed
	// to build. Only set if BuildState is FAILED.
	ErrorMessage *string
	// LogURI is the URI in S3 where the logs of this artifact's build are
	// stored. Only set if the BuildState is SUCCEEDED or (optionally) FAILED.
	LogURI *string
	// BuildType represents what kind of build is performed to generate this
	// artifact. The value here is consumed by the wrapper and determines what
	// metadata about the artifact is available to the builder.
	BuildType BuildType
	// Ingredient contains all the ingredient metadata of the ingredient being
	// built to create this artifact if this is a builder-type artifact.
	// Otherwise this field is nil.
	Ingredient *IngredientMetadata
	// Builder contains all the ingredient metadata of the builder ingredient
	// used to build this artifact.
	Builder *IngredientMetadata
	// BuildDependencies references all other artifacts which this artifact has
	// build dependencies on, meaning they must be successfully built before
	// this artifact's build can be started.
	BuildDependencies []*Artifact
	ImageType         string
	ImageName         string
	ImageVersion      string
	PlatformName      string
	PlatformID        uuid.UUID
	LastModified      time.Time
	// PatchURIs contain the S3 URIs of patches that must be applied to the
	// ingredient source code before building this ingredient. Only non-empty if
	// BuildType is BUILDER.
	PatchURIs []string
	// BuilderDependencies references ingredients in the `builder-libs`
	// namespace which the builder (but not the ingredient itself) depends on
	BuilderDependencies []*BuilderDependency
}

func (a *Artifact) postOrder(visited map[uuid.UUID]bool, order *[]*Artifact) {
	visited[a.ArtifactID] = true

	for _, dependency := range a.BuildDependencies {
		if _, ok := visited[dependency.ArtifactID]; !ok {
			dependency.postOrder(visited, order)
		}
	}

	*order = append(*order, a)
}

func (a *Artifact) initializeBuildState(existingArtifacts map[uuid.UUID]*Artifact) {
	if a.BuildState != 0 {
		// Already initialized
		return
	}

	for _, dependency := range a.BuildDependencies {
		dependency.initializeBuildState(existingArtifacts)
	}

	if existingArtifact, ok := existingArtifacts[a.ArtifactID]; ok {
		a.initializeBuildStateFrom(existingArtifact)
		return
	}

	buildState := Ready
	for _, dependency := range a.BuildDependencies {
		if dependency.BuildState == Failed ||
			dependency.BuildState == Doomed ||
			dependency.BuildState == Skipped {
			buildState = Doomed
		} else if buildState != Doomed && dependency.BuildState != Succeeded {
			buildState = Blocked
		}
	}

	a.BuildState = buildState
	if a.BuildState == Ready {
		now := DBFriendlyTimestamp()
		a.ReadyAt = &now
	}
}

func (a *Artifact) initializeBuildStateFrom(existingArtifact *Artifact) {
	a.BuildState = existingArtifact.BuildState
	a.ReadyAt = existingArtifact.ReadyAt
	a.PreviousAttempts = existingArtifact.PreviousAttempts
	a.URI = existingArtifact.URI
	a.Checksum = existingArtifact.Checksum
	a.ErrorMessage = existingArtifact.ErrorMessage
	a.LogURI = existingArtifact.LogURI
	a.LastModified = existingArtifact.LastModified
}

func (a *Artifact) String() string {
	switch a.BuildType {
	case Builder:
		return fmt.Sprintf("Build of %s", a.Ingredient.FullName())
	case Packager:
		return fmt.Sprintf("Packager %s", a.Builder.FullName())
	default:
		return "UNKNOWN ARTIFACT TYPE"
	}
}

// DBFriendlyTimestamp generates a timestamp that can be stored in the database
// in perfect precision.
func DBFriendlyTimestamp() time.Time {
	// DB only stores timestamps at microsecond precision. Truncate so that
	// a value read out of the DB is exactly equal to it's go-initialized
	// counterpart (makes writing testing assertions easier).
	return time.Now().Truncate(time.Microsecond).UTC()
}

// ArtifactSorter can be passed to sort.SliceStable to sort a slice of artifacts
// by the namespace/name/version/revision of the ingredient if a builder
// artifact, otherwise by the namespace/name/version/revision of the builder if
// a packager artifact.
func ArtifactSorter(artifacts []*Artifact) func(int, int) bool {
	return func(i, j int) bool {
		return artifacts[i].String() < artifacts[j].String()
	}
}
