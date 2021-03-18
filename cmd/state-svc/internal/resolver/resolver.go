package resolver

import (
	"golang.org/x/net/context"

	genserver "github.com/ActiveState/cli/cmd/state-svc/internal/server/generated"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/graph"
)

type Resolver struct{}

var _ genserver.ResolverRoot = &Resolver{} // Must implement ResolverRoot

func New() *Resolver {
	return &Resolver{}
}

// Seems gqlgen supplies this so you can separate your resolver and query resolver logic
// So far no need for this, so we're pointing back at ourselves..
func (r *Resolver) Query() genserver.QueryResolver { return r }

func (r *Resolver) Version(ctx context.Context) (*graph.Version, error) {
	return &graph.Version{
		&graph.StateVersion{
			License:  constants.LibraryLicense,
			Version:  constants.Version,
			Branch:   constants.BranchName,
			Revision: constants.RevisionHash,
			Date:     constants.Date,
		},
	}, nil
}
