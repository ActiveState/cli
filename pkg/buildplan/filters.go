package buildplan

import (
	"strings"

	"github.com/ActiveState/cli/internal/sliceutils"
	"github.com/ActiveState/cli/pkg/buildplan/raw"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	"github.com/go-openapi/strfmt"
)

type FilterArtifact func(a *Artifact) bool

func FilterPlatformArtifacts(platformID strfmt.UUID) FilterArtifact {
	return func(a *Artifact) bool {
		if a.platforms == nil {
			return false
		}
		return sliceutils.Contains(a.platforms, platformID)
	}
}

func FilterBuildtimeArtifacts() FilterArtifact {
	return func(a *Artifact) bool {
		return a.isBuildtimeDependency
	}
}

func FilterRuntimeArtifacts() FilterArtifact {
	return func(a *Artifact) bool {
		return a.isRuntimeDependency
	}
}

func FilterArtifactIDs(ids ...strfmt.UUID) FilterArtifact {
	filterMap := sliceutils.ToLookupMap(ids)
	return func(a *Artifact) bool {
		_, ok := filterMap[a.ArtifactID]
		return ok
	}
}

const NamespaceInternal = "internal"

func FilterStateArtifacts() FilterArtifact {
	return func(a *Artifact) bool {
		internalIngredients := sliceutils.Filter(a.Ingredients, func(i *Ingredient) bool {
			return i.Namespace == NamespaceInternal
		})
		if len(a.Ingredients) == len(internalIngredients) {
			return false
		}
		if strings.Contains(a.URL, "as-builds/noop") {
			return false
		}
		return raw.IsStateToolMimeType(a.MimeType)
	}
}

func FilterSuccessfulArtifacts() FilterArtifact {
	return func(a *Artifact) bool {
		return a.Status == types.ArtifactSucceeded ||
			a.Status == types.ArtifactBlocked ||
			a.Status == types.ArtifactStarted ||
			a.Status == types.ArtifactReady
	}
}