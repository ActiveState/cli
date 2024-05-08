package buildplanner

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/gqlclient"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildexpression"
	"github.com/ActiveState/graphql"
)

const clientDeprecationErrorKey = "CLIENT_DEPRECATION_ERROR"

type client struct {
	gqlClient *gqlclient.Client
}

func logRequestVariables(req gqlclient.Request) {
	if !strings.EqualFold(os.Getenv(constants.DebugServiceRequestsEnvVarName), "true") {
		return
	}

	vars, err := req.Vars()
	if err != nil {
		// Don't fail request because of this errors
		logging.Error("Failed to get request vars: %s", err)
		return
	}

	for _, v := range vars {
		if _, ok := v.(*buildexpression.BuildExpression); !ok {
			continue
		}

		beData, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			logging.Error("Failed to marshal build expression: %s", err)
			return
		}
		logging.Debug("Build Expression: %s", string(beData))
	}
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
