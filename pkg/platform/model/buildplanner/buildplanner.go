package buildplanner

import (
	"time"

	"github.com/ActiveState/cli/internal/gqlclient"
	"github.com/ActiveState/cli/internal/graphql"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

const clientDeprecationErrorKey = "CLIENT_DEPRECATION_ERROR"

type client struct {
	gqlClient *gqlclient.Client
}

type BuildPlanner struct {
	auth   *authentication.Auth
	client *client
	cache  cacher
}

type cacher interface {
	GetCache(key string) (string, error)
	SetCache(key, value string, expiry time.Duration) error
}

type VoidCacher struct{}

func (v VoidCacher) GetCache(key string) (string, error) {
	return "", nil
}

func (v VoidCacher) SetCache(key, value string, expiry time.Duration) error {
	return nil
}

func NewBuildPlannerModel(auth *authentication.Auth, cache cacher) *BuildPlanner {
	bpURL := api.GetServiceURL(api.ServiceBuildPlanner).String()
	logging.Debug("Using build planner at: %s", bpURL)

	gqlClient := gqlclient.NewWithOpts(bpURL, 0, graphql.WithHTTPClient(api.NewHTTPClient()))

	if auth != nil && auth.Authenticated() {
		gqlClient.SetTokenProvider(auth)
	}

	// To avoid error prone nil checks all over the place
	if cache == nil {
		cache = VoidCacher{}
	}

	return &BuildPlanner{
		auth: auth,
		client: &client{
			gqlClient: gqlClient,
		},
		cache: cache,
	}
}
