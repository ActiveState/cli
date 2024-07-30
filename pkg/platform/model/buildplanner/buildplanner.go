package buildplanner

import (
	"github.com/ActiveState/cli/internal/gqlclient"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/graphql"
)

const clientDeprecationErrorKey = "CLIENT_DEPRECATION_ERROR"

type client struct {
	gqlClient *gqlclient.Client
}

type BuildPlanner struct {
	auth   *authentication.Auth
	client *client
}

func NewBuildPlannerModel(auth *authentication.Auth) *BuildPlanner {
	bpURL := api.GetServiceURL(api.ServiceBuildPlanner).String()
	logging.Debug("Using build planner at: %s", bpURL)

	gqlClient := gqlclient.NewWithOpts(bpURL, 0, graphql.WithHTTPClient(api.NewHTTPClient()))

	if auth != nil && auth.Authenticated() {
		gqlClient.SetTokenProvider(auth)
	}

	return &BuildPlanner{
		auth: auth,
		client: &client{
			gqlClient: gqlClient,
		},
	}
}
