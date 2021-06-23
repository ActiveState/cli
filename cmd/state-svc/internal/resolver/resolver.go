package resolver

import (
	"fmt"
	"path/filepath"
	"sort"
	"time"

	"golang.org/x/net/context"

	genserver "github.com/ActiveState/cli/cmd/state-svc/internal/server/generated"
	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/graph"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/patrickmn/go-cache"
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
	logging.Debug("Version resolver")
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

func (r *Resolver) AvailableUpdate(ctx context.Context) (*graph.AvailableUpdate, error) {
	logging.Debug("AvailableUpdate resolver")
	const cacheKey = "AvailableUpdate"
	c := cache.New(12*time.Hour, time.Hour)
	if up, exists := c.Get(cacheKey); exists {
		return up.(*graph.AvailableUpdate), nil
	}

	update, err := updater.DefaultChecker.Check()
	if err != nil {
		return nil, fmt.Errorf("Failed to check for available update: %w", errs.Join(err, ": "))
	}
	if update == nil {
		return nil, nil
	}

	availableUpdate := graph.AvailableUpdate{
		Version:  update.Version,
		Channel:  update.Channel,
		Path:     update.Path,
		Platform: update.Platform,
		Sha256:   update.Sha256,
	}

	c.Set(cacheKey, &availableUpdate, cache.DefaultExpiration)

	return &availableUpdate, nil
}

func (r *Resolver) Update(ctx context.Context, channel *string, version *string) (*graph.DeferredUpdate, error) {
	logging.Debug("Update resolver")
	ch := ""
	ver := ""
	if channel != nil {
		ch = *channel
	}
	if version != nil {
		ver = *version
	}
	up, err := updater.DefaultChecker.CheckFor(ch, ver)
	if err != nil {
		return nil, fmt.Errorf("Failed to check for specified update: %w", errs.Join(err, ": "))
	}
	if up == nil {
		return &graph.DeferredUpdate{}, nil
	}
	installTargetPath := filepath.Dir(appinfo.StateApp().Exec())
	proc, err := up.InstallDeferred(installTargetPath)
	if err != nil {
		return nil, fmt.Errorf("Deferring update failed: %w", errs.Join(err, ": "))
	}

	return &graph.DeferredUpdate{
		Channel: up.Channel,
		Version: up.Version,
		Logfile: installation.LogfilePath(r.cfg.ConfigPath(), proc.Pid),
	}, nil
}

func (r *Resolver) Projects(ctx context.Context) ([]*graph.Project, error) {
	logging.Debug("Projects resolver")
	if err := r.cfg.Reload(); err != nil {
		return nil, errs.Wrap(err, "failed to reload configuration")
	}

	var projects []*graph.Project
	localConfigProjects := projectfile.GetProjectMapping(r.cfg)
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
