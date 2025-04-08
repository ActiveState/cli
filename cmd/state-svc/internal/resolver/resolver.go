package resolver

import (
	"context"
	"encoding/json"
	"os"
	"runtime/debug"
	"sort"
	"strconv"
	"time"

	"github.com/ActiveState/cli/cmd/state-svc/internal/graphqltypes"
	"github.com/ActiveState/cli/cmd/state-svc/internal/hash"
	"github.com/ActiveState/cli/cmd/state-svc/internal/notifications"
	"github.com/ActiveState/cli/cmd/state-svc/internal/rtwatcher"
	genserver "github.com/ActiveState/cli/cmd/state-svc/internal/server/generated"
	"github.com/ActiveState/cli/internal/analytics/client/sync"
	anaConsts "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"github.com/ActiveState/cli/internal/cache/projectcache"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/graph"
	"github.com/ActiveState/cli/internal/logging"
	configMediator "github.com/ActiveState/cli/internal/mediators/config"
	"github.com/ActiveState/cli/internal/poller"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/runbits/panics"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/patrickmn/go-cache"
)

type Resolver struct {
	cfg            *config.Instance
	messages       *notifications.Notifications
	updatePoller   *poller.Poller
	authPoller     *poller.Poller
	projectIDCache *projectcache.ID
	fileHasher     *hash.FileHasher
	an             *sync.Client
	anForClient    *sync.Client // Use separate client for events sent through service so we don't contaminate one with the other
	rtwatch        *rtwatcher.Watcher
	auth           *authentication.Auth
	globalCache    *cache.Cache
}

// var _ genserver.ResolverRoot = &Resolver{} // Must implement ResolverRoot

func New(cfg *config.Instance, an *sync.Client, auth *authentication.Auth) (*Resolver, error) {
	msg, err := notifications.New(cfg, auth)
	if err != nil {
		return nil, errs.Wrap(err, "Could not initialize messages")
	}

	upchecker := updater.NewDefaultChecker(cfg, an)
	pollUpdate := poller.New(1*time.Hour, func() (interface{}, error) {
		defer func() {
			panics.LogAndPanic(recover(), debug.Stack())
		}()
		logging.Debug("Poller checking for update info")
		return upchecker.CheckFor(constants.ChannelName, "")
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
		defer func() {
			panics.LogAndPanic(recover(), debug.Stack())
		}()
		if auth.SyncRequired() {
			return nil, auth.Sync()
		}
		return nil, nil
	})

	// Note: source does not matter here, as analytics sent via the resolver have a source
	// (e.g. State Tool or Executor), and that source will be used.
	anForClient := sync.New(anaConsts.SrcStateTool, cfg, auth, nil)
	return &Resolver{
		cfg,
		msg,
		pollUpdate,
		pollAuth,
		projectcache.NewID(),
		hash.NewFileHasher(),
		an,
		anForClient,
		rtwatcher.New(cfg, anForClient),
		auth,
		cache.New(time.Hour, 10*time.Minute),
	}, nil
}

func (r *Resolver) Close() error {
	r.messages.Close()
	r.updatePoller.Close()
	r.authPoller.Close()
	r.anForClient.Close()
	return r.rtwatch.Close()
}

// Seems gqlgen supplies this so you can separate your resolver and query resolver logic
// So far no need for this, so we're pointing back at ourselves..
func (r *Resolver) Query() genserver.QueryResolver { return r }

func (r *Resolver) Mutation() genserver.MutationResolver { return r }

func (r *Resolver) Version(ctx context.Context) (*graph.Version, error) {
	defer func() { panics.LogAndPanic(recover(), debug.Stack()) }()

	r.an.EventWithLabel(anaConsts.CatStateSvc, "endpoint", "Version")
	logging.Debug("Version resolver")
	return &graph.Version{
		State: &graph.StateVersion{
			License:  constants.LibraryLicense,
			Version:  constants.Version,
			Channel:  constants.ChannelName,
			Revision: constants.RevisionHash,
			Date:     constants.Date,
		},
	}, nil
}

func (r *Resolver) AvailableUpdate(ctx context.Context, desiredChannel, desiredVersion string) (*graph.AvailableUpdate, error) {
	defer func() { panics.LogAndPanic(recover(), debug.Stack()) }()

	if desiredChannel == "" {
		desiredChannel = constants.ChannelName
	}

	r.an.EventWithLabel(anaConsts.CatStateSvc, "endpoint", "AvailableUpdate")
	logging.Debug("AvailableUpdate resolver: %s/%s", desiredChannel, desiredVersion)
	defer logging.Debug("AvailableUpdate done")

	var (
		avUpdate *updater.AvailableUpdate
		ok       bool
		err      error
	)

	switch {
	case desiredChannel == constants.ChannelName && desiredVersion == "":
		avUpdate, ok = r.updatePoller.ValueFromCache().(*updater.AvailableUpdate)
		if !ok || avUpdate == nil {
			logging.Debug("No update info in poller cache")
			return nil, nil
		}

		logging.Debug("Update info pulled from poller cache")

	default:
		logging.Debug("Update info requested for specific channel/version")

		upchecker := updater.NewDefaultChecker(r.cfg, r.an)
		avUpdate, err = upchecker.CheckFor(desiredChannel, desiredVersion)
		if err != nil {
			return nil, errs.Wrap(err, "Failed to check for specified channel/version: %s/%s", desiredChannel, desiredVersion)
		}
	}

	availableUpdate := &graph.AvailableUpdate{
		Version:  avUpdate.Version,
		Channel:  avUpdate.Channel,
		Path:     avUpdate.Path,
		Platform: avUpdate.Platform,
		Sha256:   avUpdate.Sha256,
	}

	return availableUpdate, nil
}

func (r *Resolver) Projects(ctx context.Context) ([]*graph.Project, error) {
	defer func() { panics.LogAndPanic(recover(), debug.Stack()) }()

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

func (r *Resolver) AnalyticsEvent(_ context.Context, category, action, source string, _label *string, dimensionsJson string) (*graph.AnalyticsEventResponse, error) {
	defer func() { panics.LogAndPanic(recover(), debug.Stack()) }()

	logging.Debug("Analytics event resolver: %s - %s: %s (%s)", category, action, ptr.From(_label, "NIL"), source)

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
		id, err := r.projectIDCache.FromNamespace(*values.ProjectNameSpace, r.auth)
		if err != nil {
			logging.Error("Could not resolve project ID for analytics: %s", errs.JoinMessage(err))
		}
		values.ProjectID = &id
		return nil
	})

	r.anForClient.EventWithSourceAndLabel(category, action, source, label, dims)

	return &graph.AnalyticsEventResponse{Sent: true}, nil
}

func (r *Resolver) ReportRuntimeUsage(_ context.Context, pid int, exec, source string, dimensionsJSON string) (*graph.ReportRuntimeUsageResponse, error) {
	defer func() { panics.LogAndPanic(recover(), debug.Stack()) }()

	logging.Debug("Runtime usage resolver: %d - %s", pid, exec)
	var dims *dimensions.Values
	if err := json.Unmarshal([]byte(dimensionsJSON), &dims); err != nil {
		return &graph.ReportRuntimeUsageResponse{Received: false}, errs.Wrap(err, "Could not unmarshal")
	}

	r.rtwatch.Watch(pid, exec, source, dims)

	return &graph.ReportRuntimeUsageResponse{Received: true}, nil
}

func (r *Resolver) CheckNotifications(ctx context.Context, command string, flags []string) ([]*graph.NotificationInfo, error) {
	defer func() { panics.LogAndPanic(recover(), debug.Stack()) }()
	logging.Debug("Check notifications resolver")
	return r.messages.Check(command, flags)
}

func (r *Resolver) ConfigChanged(ctx context.Context, key string) (*graph.ConfigChangedResponse, error) {
	defer func() { panics.LogAndPanic(recover(), debug.Stack()) }()

	go configMediator.NotifyListeners(key)
	return &graph.ConfigChangedResponse{Received: true}, nil
}

func (r *Resolver) FetchLogTail(ctx context.Context) (string, error) {
	defer func() { panics.LogAndPanic(recover(), debug.Stack()) }()

	return logging.ReadTail(), nil
}

func (r *Resolver) GetProcessesInUse(ctx context.Context, execDir string) ([]*graph.ProcessInfo, error) {
	defer func() { panics.LogAndPanic(recover(), debug.Stack()) }()

	inUse := r.rtwatch.GetProcessesInUse(execDir)
	processes := make([]*graph.ProcessInfo, 0, len(inUse))
	for _, entry := range inUse {
		processes = append(processes, &graph.ProcessInfo{entry.Exec, entry.PID})
	}
	return processes, nil
}

func (r *Resolver) GetJwt(ctx context.Context) (*graph.Jwt, error) {
	defer func() { panics.LogAndPanic(recover(), debug.Stack()) }()

	if err := r.auth.MaybeRenew(); err != nil {
		return nil, errs.Wrap(err, "Could not renew auth token")
	}

	if !r.auth.Authenticated() {
		return nil, nil
	}

	user := r.auth.User()
	if user == nil {
		return nil, errs.New("user is nil")
	}

	jwt := &graph.Jwt{
		Token: r.auth.BearerToken(),
		User: &graph.User{
			UserID:        user.UserID.String(),
			Username:      user.Username,
			Email:         user.Email,
			Organizations: []*graph.Organization{},
		},
	}

	for _, org := range user.Organizations {
		jwt.User.Organizations = append(jwt.User.Organizations, &graph.Organization{
			URLname: org.URLname,
			Role:    org.Role,
		})
	}

	return jwt, nil
}

func (r *Resolver) HashGlobs(ctx context.Context, wd string, globs []string) (*graph.GlobResult, error) {
	defer func() { panics.LogAndPanic(recover(), debug.Stack()) }()

	hash, files, err := r.fileHasher.HashFiles(wd, globs)
	if err != nil {
		return nil, errs.Wrap(err, "Could not hash files")
	}

	result := &graph.GlobResult{
		Hash: hash,
	}
	for _, f := range files {
		result.Files = append(result.Files, &graph.GlobFileResult{
			Pattern: f.Pattern,
			Path:    f.Path,
			Hash:    f.Hash,
		})
	}

	return result, nil
}

func (r *Resolver) GetCache(ctx context.Context, key string) (string, error) {
	v, exists := r.globalCache.Get(key)
	if !exists {
		return "", nil
	}
	return v.(string), nil
}

func (r *Resolver) SetCache(ctx context.Context, key string, value string, expiry int) (*graphqltypes.Void, error) {
	r.globalCache.Set(key, value, time.Duration(expiry)*time.Second)
	return &graphqltypes.Void{}, nil
}
