package resolver

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/ActiveState/cli/cmd/state-svc/internal/rtwatcher"
	"github.com/ActiveState/cli/internal/analytics/client/sync"
	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"github.com/ActiveState/cli/internal/cache/projectcache"
	"golang.org/x/net/context"

	genserver "github.com/ActiveState/cli/cmd/state-svc/internal/server/generated"
	anaConsts "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/graph"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/patrickmn/go-cache"
)

type Resolver struct {
	cfg            *config.Instance
	cache          *cache.Cache
	projectIDCache *projectcache.ID
	an             *sync.Client
	rtwatch        *rtwatcher.Watcher
}

// var _ genserver.ResolverRoot = &Resolver{} // Must implement ResolverRoot

func New(cfg *config.Instance, an *sync.Client) *Resolver {
	return &Resolver{
		cfg,
		cache.New(12*time.Hour, time.Hour),
		projectcache.NewID(),
		an,
		rtwatcher.New(cfg, an),
	}
}

func (r *Resolver) Close() error {
	return r.rtwatch.Close()
}

// Seems gqlgen supplies this so you can separate your resolver and query resolver logic
// So far no need for this, so we're pointing back at ourselves..
func (r *Resolver) Query() genserver.QueryResolver { return r }

func (r *Resolver) Version(ctx context.Context) (*graph.Version, error) {
	r.an.EventWithLabel(anaConsts.CatStateSvc, "endpoint", "Version")
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
	r.an.EventWithLabel(anaConsts.CatStateSvc, "endpoint", "AvailableUpdate")
	logging.Debug("AvailableUpdate resolver")
	defer logging.Debug("AvailableUpdate done")

	const cacheKey = "AvailableUpdate"
	if up, exists := r.cache.Get(cacheKey); exists {
		logging.Debug("Using cache")
		return up.(*graph.AvailableUpdate), nil
	}

	var availableUpdate *graph.AvailableUpdate
	defer func() { r.cache.Set(cacheKey, availableUpdate, cache.DefaultExpiration) }()

	update, err := updater.NewDefaultChecker(r.cfg).Check()
	if err != nil {
		return nil, fmt.Errorf("Failed to check for available update: %w", errs.Join(err, ": "))
	}
	if update == nil {
		return nil, nil
	}

	availableUpdate = &graph.AvailableUpdate{
		Version:  update.Version,
		Channel:  update.Channel,
		Path:     update.Path,
		Platform: update.Platform,
		Sha256:   update.Sha256,
	}

	return availableUpdate, nil
}

func (r *Resolver) Projects(ctx context.Context) ([]*graph.Project, error) {
	r.an.EventWithLabel(anaConsts.CatStateSvc, "endpoint", "Projects")
	logging.Debug("Projects resolver")
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

func (r *Resolver) AnalyticsEvent(_ context.Context, category, action string, _label *string, dimensionsJson string) (*graph.AnalyticsEventResponse, error) {
	logging.Debug("Analytics event resolver")

	label := ""
	if _label != nil {
		label = *_label
	}

	var dims *dimensions.Values
	if err := json.Unmarshal([]byte(dimensionsJson), &dims); err != nil {
		return &graph.AnalyticsEventResponse{Sent: false}, errs.Wrap(err, "Could not unmarshal")
	}

	// Resolve the project ID - this is a little awkward since I had to work around an import cycle
	dims.RegisterPreProcessor(func(values *dimensions.Values) error {
		values.ProjectID = nil
		if values.ProjectNameSpace == nil {
			return nil
		}
		id, err := r.projectIDCache.FromNamespace(*values.ProjectNameSpace)
		if err != nil {
			return errs.Wrap(err, "Could not resolve project ID")
		}
		values.ProjectID = &id
		return nil
	})

	r.an.EventWithLabel(category, action, label, dims)

	return &graph.AnalyticsEventResponse{Sent: true}, nil
}

func (r *Resolver) RuntimeUsage(ctx context.Context, pid int, exec string, dimensionsJSON string) (*graph.RuntimeUsageResponse, error) {
	var dims *dimensions.Values
	if err := json.Unmarshal([]byte(dimensionsJSON), &dims); err != nil {
		return &graph.RuntimeUsageResponse{Received: false}, errs.Wrap(err, "Could not unmarshal")
	}

	r.rtwatch.Watch(pid, exec, dims)

	return &graph.RuntimeUsageResponse{Received: true}, nil
}
