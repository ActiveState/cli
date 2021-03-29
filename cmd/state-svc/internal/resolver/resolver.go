package resolver

import (
	"sort"

	"golang.org/x/net/context"

	genserver "github.com/ActiveState/cli/cmd/state-svc/internal/server/generated"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/graph"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type Resolver struct{}

// var _ genserver.ResolverRoot = &Resolver{} // Must implement ResolverRoot

func New() *Resolver {
	return &Resolver{}
}

// Seems gqlgen supplies this so you can separate your resolver and query resolver logic
// So far no need for this, so we're pointing back at ourselves..
func (r *Resolver) Query() genserver.QueryResolver { return r }

func (r *Resolver) Version(ctx context.Context) (*graph.Version, error) {
	logging.Debug("Version resolver")
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

func (r *Resolver) Projects(ctx context.Context) ([]*graph.Project, error) {
	logging.Debug("Projects resolver")
	config, err := config.New()
	if err != nil {
		return nil, locale.WrapError(err, "err_resolver_get_config", "Could not get new config instance")
	}

	var projects []*graph.Project
	localConfigProjects := config.GetStringMapStringSlice(projectfile.LocalProjectsConfigKey)
	for ns, locations := range localConfigProjects {
		projects = append(projects, &graph.Project{
			Namespace: ns,
			Locations: locations,
		})
	}
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Namespace < projects[j].Namespace
	})

	return projects, nil
}
