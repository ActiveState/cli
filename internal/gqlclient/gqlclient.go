package gqlclient

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/machinebox/graphql"

	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/machineid"
	"github.com/ActiveState/cli/internal/retryhttp"
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

type Client struct {
	*graphqlClient
	tokenProvider BearerTokenProvider
	timeout       time.Duration
}

func New(url string, timeout time.Duration) *Client {
	if timeout == 0 {
		timeout = time.Second * 60
	}

	retryOpt := graphql.WithHTTPClient(retryhttp.DefaultClient.StandardClient())

	client := &Client{
		graphqlClient: graphql.NewClient(url, retryOpt),
		timeout:       timeout,
	}
	client.graphqlClient.Log = func(s string) { logging.Debug("Third party log message: %s", s) }
	return client
}

func (c *Client) SetTokenProvider(tokenProvider BearerTokenProvider) {
	c.tokenProvider = tokenProvider
}

func (c *Client) SetDebug(b bool) {
	c.graphqlClient.Log = func(string) {}
	if b {
		c.graphqlClient.Log = func(s string) {
			fmt.Fprintln(os.Stderr, s)
		}
	}
}

func (c *Client) Run(request Request, response interface{}) error {
	graphRequest := graphql.NewRequest(request.Query())
	for key, value := range request.Vars() {
		graphRequest.Var(key, value)
	}

	ctx := context.Background()
	var cancel context.CancelFunc

	ctx, cancel = context.WithTimeout(ctx, c.timeout)
	defer cancel()

	var bearerToken string
	if c.tokenProvider != nil {
		bearerToken = c.tokenProvider.BearerToken()
		if bearerToken != "" {
			graphRequest.Header.Set("Authorization", "Bearer "+bearerToken)
		}
	}

	graphRequest.Header.Set("X-Requestor", machineid.UniqID())

	return c.graphqlClient.Run(ctx, graphRequest, &response)
}
