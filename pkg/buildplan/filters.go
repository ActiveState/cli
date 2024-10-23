package buildplan

import (
	"os"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
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
		internalIngredients := []*Ingredient{}
		if os.Getenv(constants.InstallInternalDependenciesEnvVarName) != "true" {
			internalIngredients = sliceutils.Filter(a.Ingredients, func(i *Ingredient) bool {
				return i.Namespace == NamespaceInternal
			})
		}
		if len(a.Ingredients) > 0 && len(a.Ingredients) == len(internalIngredients) {
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
		return a.Status == types.ArtifactSucceeded
	}
}

func FilterFailedArtifacts() FilterArtifact {
	return func(a *Artifact) bool {
		return a.Status == types.ArtifactFailedTransiently ||
			a.Status == types.ArtifactFailedPermanently
	}
}

func FilterNotBuild() FilterArtifact {
	return func(a *Artifact) bool {
		return a.Status != types.ArtifactSucceeded
	}
}

type FilterOutIngredients struct {
	Ingredients IngredientIDMap
}

func (f FilterOutIngredients) Filter(i *Ingredient) bool {
	_, blacklist := f.Ingredients[i.IngredientID]
	return !blacklist
}
