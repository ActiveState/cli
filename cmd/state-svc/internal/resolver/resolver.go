package resolver

import (
	"golang.org/x/net/context"

	genserver "github.com/ActiveState/cli/cmd/state-svc/internal/server/generated"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/graph"
	"github.com/ActiveState/cli/internal/updater2"
)

type Resolver struct {
	cfg *config.Instance
}

// var _ genserver.ResolverRoot = &Resolver{} // Must implement ResolverRoot

func New(cfg *config.Instance) *Resolver {
	return &Resolver{cfg}
}

// Seems gqlgen supplies this so you can separate your resolver and query resolver logic
// So far no need for this, so we're pointing back at ourselves..
func (r *Resolver) Query() genserver.QueryResolver { return r }

func (r *Resolver) Version(ctx context.Context) (*graph.Version, error) {
	return &graph.Version{
		State: &graph.StateVersion{
			License:  constants.LibraryLicense,
			Version:  constants.Version,
			Branch:   constants.BranchName,
			Revision: constants.RevisionHash,
			Date:     constants.Date,
		},
	}, nil
}

func (r *Resolver) Update(ctx context.Context, channel *string, version *string) (*graph.DeferredUpdate, error) {
	ch := ""
	ver := ""
	if channel != nil {
		ch = *channel
	}
	if version != nil {
		ver = *version
	}
	up, err := updater2.DefaultChecker.CheckFor(ch, ver)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to check for update")
	}
	if up == nil {
		return &graph.DeferredUpdate{}, nil
	}
	err = up.InstallDeferred(r.cfg.ConfigPath())
	if err != nil {
		return nil, errs.Wrap(err, "Deferring update failed")
	}

	return &graph.DeferredUpdate{
		Channel: up.Channel(),
		Version: up.Version(),
	}, nil
}
