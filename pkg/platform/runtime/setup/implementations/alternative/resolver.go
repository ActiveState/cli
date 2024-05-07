package alternative

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/buildplan"
	"github.com/go-openapi/strfmt"
)

type Resolver struct {
	artifactsForNameResolving buildplan.ArtifactIDMap
}

func NewResolver(artifactsForNameResolving buildplan.ArtifactIDMap) *Resolver {
	return &Resolver{artifactsForNameResolving: artifactsForNameResolving}
}

func (r *Resolver) ResolveArtifactName(id strfmt.UUID) string {
	if artf, ok := r.artifactsForNameResolving[id]; ok {
		return artf.Name()
	}
	return locale.T("alternative_unknown_pkg_name")
}
