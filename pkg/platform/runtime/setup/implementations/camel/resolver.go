package camel

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/go-openapi/strfmt"
)

type Resolver struct{}

func NewResolver() *Resolver {
	return &Resolver{}
}

func (r *Resolver) ResolveArtifactName(_ strfmt.UUID) string {
	return locale.Tl("camel_bundle_name", "legacy bundle")
}
