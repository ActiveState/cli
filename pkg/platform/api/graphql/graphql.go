package graphql

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/retryhttp"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/authentication"

	"github.com/machinebox/graphql"
)

type Request interface {
	Query() string
	Vars() map[string]interface{}
}

type Header map[string][]string

type graphqlRequest = graphql.Request

type graphqlClient = graphql.Client

type BearerTokenProvider interface {
	BearerToken() string
}

type GQLClient struct {
	*graphqlClient
	common        Header
	tokenProvider BearerTokenProvider
	timeout       time.Duration
}

// Get is a legacy method, to be removed once we have commands that don't rely on globals
func Get() *GQLClient {
	url := api.GetServiceURL(api.ServiceGraphQL)
	return New(url.String(), map[string][]string{}, authentication.Get(), 0)
}

func New(url string, common Header, bearerToken BearerTokenProvider, timeout time.Duration) *GQLClient {
	defClient := retryhttp.DefaultClient
	timeout = defClient.MaxTimeout(retryhttp.DefaultTimeout, timeout, time.Second*60)

	retryOpt := graphql.WithHTTPClient(defClient.StandardClient())

	return &GQLClient{
		graphqlClient: graphql.NewClient(url, retryOpt),
		common:        common,
		tokenProvider: bearerToken,
		timeout:       timeout,
	}
}

func (c *GQLClient) SetDebug(b bool) {
	c.graphqlClient.Log = func(string) {}
	if b {
		c.graphqlClient.Log = func(s string) {
			fmt.Fprintln(os.Stderr, s)
		}
	}
}

func (c *GQLClient) Run(request Request, response interface{}) error {
	graphRequest := graphql.NewRequest(request.Query())
	for key, value := range request.Vars() {
		graphRequest.Var(key, value)
	}

	ctx := context.Background()
	var cancel context.CancelFunc

	ctx, cancel = context.WithTimeout(ctx, c.timeout)
	defer cancel()

	bearerToken := c.tokenProvider.BearerToken()
	if bearerToken != "" {
		graphRequest.Header.Set("Authorization", "Bearer "+bearerToken)
	}

	graphRequest.Header.Set("X-Requestor", logging.UniqID())

	return c.graphqlClient.Run(ctx, graphRequest, response)
}
