package resolver

import (
	"encoding/json"
	"os"
	"runtime/debug"
	"sort"
	"strconv"
	"time"

	"github.com/ActiveState/cli/cmd/state-svc/internal/deprecation"
	"github.com/ActiveState/cli/cmd/state-svc/internal/rtusage"
	"github.com/ActiveState/cli/cmd/state-svc/internal/rtwatcher"
	"github.com/ActiveState/cli/internal/analytics/client/sync"
	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"github.com/ActiveState/cli/internal/cache/projectcache"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/poller"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"golang.org/x/net/context"

	genserver "github.com/ActiveState/cli/cmd/state-svc/internal/server/generated"
	anaConsts "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/graph"
	"github.com/ActiveState/cli/internal/logging"
	configMediator "github.com/ActiveState/cli/internal/mediators/config"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type Resolver struct {
	cfg            *config.Instance
	depPoller      *poller.Poller
	updatePoller   *poller.Poller
	authPoller     *poller.Poller
	usageChecker   *rtusage.Checker
	projectIDCache *projectcache.ID
	an             *sync.Client
	anForClient    *sync.Client // Use separate client for events sent through service so we don't contaminate one with the other
	rtwatch        *rtwatcher.Watcher
}

// var _ genserver.ResolverRoot = &Resolver{} // Must implement ResolverRoot

func New(cfg *config.Instance, an *sync.Client, auth *authentication.Auth) (*Resolver, error) {
	depchecker := deprecation.NewChecker(cfg)
	pollDep := poller.New(1*time.Hour, func() (interface{}, error) {
		return depchecker.Check()
	})

	upchecker := updater.NewDefaultChecker(cfg)
	pollUpdate := poller.New(1*time.Hour, func() (interface{}, error) {
		return upchecker.Check()
	})

	pollRate := time.Minute.Milliseconds()
	if override := os.Getenv(constants.SvcAuthPollingRateEnvVarName); override != "" {
		overrideInt, err := strconv.ParseInt(override, 10, 64)
		if err != nil {
			return nil, errs.New("Failed to parse svc polling time override: %v", err)
		}
		pollRate = overrideInt
	}

	pollAuth := poller.New(time.Duration(int64(time.Millisecond)*pollRate), func() (interface{}, error) {
		if auth.SyncRequired() {
			return nil, auth.Sync()
		}
		return nil, nil
	})

	usageChecker := rtusage.NewChecker(cfg, auth)

	anForClient := sync.New(cfg, auth)
	return &Resolver{
		cfg,
		pollDep,
		pollUpdate,
		pollAuth,
		usageChecker,
		projectcache.NewID(),
		an,
		anForClient,
		rtwatcher.New(cfg, anForClient),
	}, nil
}

func (r *Resolver) Close() error {
	r.depPoller.Close()
	r.updatePoller.Close()
	r.authPoller.Close()
	r.anForClient.Close()
	return r.rtwatch.Close()
}

// Seems gqlgen supplies this so you can separate your resolver and query resolver logic
// So far no need for this, so we're pointing back at ourselves..
func (r *Resolver) Query() genserver.QueryResolver { return r }

func (r *Resolver) Version(ctx context.Context) (*graph.Version, error) {
	defer func() { handlePanics(recover(), debug.Stack()) }()

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
	defer func() { handlePanics(recover(), debug.Stack()) }()

	r.an.EventWithLabel(anaConsts.CatStateSvc, "endpoint", "AvailableUpdate")
	logging.Debug("AvailableUpdate resolver")
	defer logging.Debug("AvailableUpdate done")

	update, ok := r.updatePoller.ValueFromCache().(*updater.AvailableUpdate)
	if !ok || update == nil {
		logging.Debug("No update info in cache")
		return nil, nil
	}

	availableUpdate := &graph.AvailableUpdate{
		Version:  update.Version,
		Channel:  update.Channel,
		Path:     update.Path,
		Platform: update.Platform,
		Sha256:   update.Sha256,
	}

	return availableUpdate, nil
}

func (r *Resolver) Projects(ctx context.Context) ([]*graph.Project, error) {
	defer func() { handlePanics(recover(), debug.Stack()) }()

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
	defer func() { handlePanics(recover(), debug.Stack()) }()

	logging.Debug("Analytics event resolver: %s - %s", category, action)

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
		if values.ProjectNameSpace == nil || *values.ProjectNameSpace == "" {
			return nil
		}
		id, err := r.projectIDCache.FromNamespace(*values.ProjectNameSpace)
		if err != nil {
			logging.Error("Could not resolve project ID for analytics: %s", errs.JoinMessage(err))
		}
		values.ProjectID = &id
		return nil
	})

	r.anForClient.EventWithLabel(category, action, label, dims)

	return &graph.AnalyticsEventResponse{Sent: true}, nil
}

func (r *Resolver) ReportRuntimeUsage(_ context.Context, pid int, exec string, dimensionsJSON string) (*graph.ReportRuntimeUsageResponse, error) {
	defer func() { handlePanics(recover(), debug.Stack()) }()

	logging.Debug("Runtime usage resolver: %d - %s", pid, exec)
	var dims *dimensions.Values
	if err := json.Unmarshal([]byte(dimensionsJSON), &dims); err != nil {
		return &graph.ReportRuntimeUsageResponse{Received: false}, errs.Wrap(err, "Could not unmarshal")
	}

	r.rtwatch.Watch(pid, exec, dims)

	return &graph.ReportRuntimeUsageResponse{Received: true}, nil
}

func (r *Resolver) CheckRuntimeUsage(_ context.Context, organizationName string) (*graph.CheckRuntimeUsageResponse, error) {
	defer func() { handlePanics(recover(), debug.Stack()) }()

	logging.Debug("CheckRuntimeUsage resolver")

	usage, err := r.usageChecker.Check(organizationName)
	if err != nil {
		return nil, errs.Wrap(err, "Could not check runtime usage: %s", errs.JoinMessage(err))
	}

	if usage == nil {
		return &graph.CheckRuntimeUsageResponse{
			Limit: 0,
			Usage: 0,
		}, nil
	}

	return &graph.CheckRuntimeUsageResponse{
		Limit: int(usage.LimitDynamicRuntimes),
		Usage: int(usage.ActiveDynamicRuntimes),
	}, nil
}

func (r *Resolver) CheckDeprecation(ctx context.Context) (*graph.DeprecationInfo, error) {
	defer func() { handlePanics(recover(), debug.Stack()) }()

	logging.Debug("Check deprecation resolver")

	deprecated, ok := r.depPoller.ValueFromCache().(*graph.DeprecationInfo)
	if !ok {
		logging.Debug("No deprecation info in cache")
	}

	return deprecated, nil
}

func (r *Resolver) ConfigChanged(ctx context.Context, key string) (*graph.ConfigChangedResponse, error) {
	defer func() { handlePanics(recover(), debug.Stack()) }()

	go configMediator.NotifyListeners(key)
	return &graph.ConfigChangedResponse{Received: true}, nil
}

func (r *Resolver) FetchLogTail(ctx context.Context) (string, error) {
	defer func() { handlePanics(recover(), debug.Stack()) }()

	return logging.ReadTail(), nil
}

func handlePanics(recovered interface{}, stack []byte) {
	if recovered != nil {
		multilog.Error("Panic: %v", recovered)
		logging.Debug("Stack: %s", string(stack))
		panic(recovered) // We're only logging the panic, not interrupting it
	}
}
