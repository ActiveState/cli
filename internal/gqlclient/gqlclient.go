package gqlclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/profile"
	"github.com/ActiveState/cli/internal/strutils"
	"github.com/machinebox/graphql"

	hsgraphql "github.com/hasura/go-graphql-client"

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
	baseUrl            string
	queryClient        *graphqlClient
	subscriptionClient *hsgraphql.SubscriptionClient
	tokenProvider      BearerTokenProvider
	timeout            time.Duration
}

type ClientOption func(*Client)

func NewWithOpts(baseUrl string, timeout time.Duration, opts ...ClientOption) *Client {
	if timeout == 0 {
		timeout = time.Second * 60
	}

	client := &Client{
		baseUrl:     baseUrl,
		queryClient: graphql.NewClient(baseUrl),
		timeout:     timeout,
	}

	for _, opt := range opts {
		opt(client)
	}
	client.queryClient.Log = func(s string) { logging.Debug("graphqlClient log message: %s", s) }

	return client
}

func New(url string, timeout time.Duration) *Client {
	return NewWithOpts(url, timeout, WithHTTPClient(retryhttp.DefaultClient.StandardClient()))
}

func WithHTTPClient(httpclient *http.Client) ClientOption {
	return func(c *Client) {
		c.queryClient = graphql.NewClient(fmt.Sprintf("%s/query", c.baseUrl), graphql.WithHTTPClient(httpclient))
	}
}

func WithSubscriptions(subUrl string) ClientOption {
	return func(c *Client) {
		c.subscriptionClient = hsgraphql.NewSubscriptionClient(subUrl)
	}
}

func (c *Client) SetTokenProvider(tokenProvider BearerTokenProvider) {
	c.tokenProvider = tokenProvider
}

func (c *Client) SetDebug(b bool) {
	c.queryClient.Log = func(string) {}
	if b {
		c.queryClient.Log = func(s string) {
			fmt.Fprintln(os.Stderr, s)
		}
	}
}

func (c *Client) RunQuery(request Request, response interface{}) error {
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

	graphRequest.Header.Set("X-Requestor", machineid.UniqID())

	return c.queryClient.Run(ctx, graphRequest, &response)
}

func (c *Client) RunSubscription(ctx context.Context, response interface{}) (chan interface{}, error) {
	result := make(chan interface{})
	_, err := c.subscriptionClient.Subscribe(response, nil, func(message *json.RawMessage, err error) error {
		if err != nil {
			return nil
		}

		err = json.Unmarshal(*message, response)
		result <- response
		return nil
	})
	if err != nil {
		return nil, errs.Wrap(err, "Could not subscribe")
	}

	go c.subscriptionClient.Run()

	return result, nil
}

func (c *Client) CloseSubscriptions() error {
	return c.subscriptionClient.Close()
}
