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
		if a.Platforms == nil {
			return false
		}
		return sliceutils.Contains(a.Platforms, platformID)
	}
}

func FilterBuildtimeArtifacts() FilterArtifact {
	return func(a *Artifact) bool {
		return a.IsBuildtimeDependency
	}
}

func FilterRuntimeArtifacts() FilterArtifact {
	return func(a *Artifact) bool {
		return a.IsRuntimeDependency
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
		for _, i := range a.Ingredients {
			if i.Namespace == NamespaceInternal {
				return false
			}
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
