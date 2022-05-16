package gqlclient

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/profile"
	"github.com/ActiveState/cli/internal/strutils"
	"github.com/machinebox/graphql"

	"github.com/ActiveState/cli/internal/retryhttp"
	"github.com/ActiveState/cli/internal/singleton/uniqid"
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

func NewWithOpts(url string, timeout time.Duration, opts ...graphql.ClientOption) *Client {
	if timeout == 0 {
		timeout = time.Second * 60
	}

	client := &Client{
		graphqlClient: graphql.NewClient(url, opts...),
		timeout:       timeout,
	}
	if os.Getenv(constants.DebugServiceRequestsEnvVarName) == "true" {
		client.EnableDebugLog()
	}
	return client
}

func New(url string, timeout time.Duration) *Client {
	return NewWithOpts(url, timeout, graphql.WithHTTPClient(retryhttp.DefaultClient.StandardClient()))
}

// EnableDebugLog turns on debug logging
func (c *Client) EnableDebugLog() {
	c.graphqlClient.Log = func(s string) { logging.Debug("graphqlClient log message: %s", s) }
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
	ctx := context.Background()
	if c.timeout != 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}
	err := c.RunWithContext(ctx, request, response)
	return err // Needs var so the cancel defer triggers at the right time
}

func (c *Client) RunWithContext(ctx context.Context, request Request, response interface{}) error {
	name := strutils.Summarize(request.Query(), 25)
	defer profile.Measure(fmt.Sprintf("gqlclient:RunWithContext:(%s)", name), time.Now())
	graphRequest := graphql.NewRequest(request.Query())
	for key, value := range request.Vars() {
		graphRequest.Var(key, value)
	}

	var bearerToken string
	if c.tokenProvider != nil {
		bearerToken = c.tokenProvider.BearerToken()
		if bearerToken != "" {
			graphRequest.Header.Set("Authorization", "Bearer "+bearerToken)
		}
	}

	graphRequest.Header.Set("X-Requestor", uniqid.Text())

	return c.graphqlClient.Run(ctx, graphRequest, &response)
}
