package alternative

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
)

type Resolver struct {
	artifactsForNameResolving artifact.Map
}

func NewResolver(artifactsForNameResolving artifact.Map) *Resolver {
	return &Resolver{artifactsForNameResolving: artifactsForNameResolving}
}

func (r *Resolver) ResolveArtifactName(id artifact.ArtifactID) string {
	if artf, ok := r.artifactsForNameResolving[id]; ok {
		return artf.Name
	}
	return locale.T("alternative_unknown_pkg_name")
}
