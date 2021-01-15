package mediator

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ActiveState/cli/internal/machineid"
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

type BearerTokenProvider interface {
	BearerToken() string
}

type MediatorClient struct {
	*graphql.Client
	common        Header
	tokenProvider BearerTokenProvider
	timeout       time.Duration
}

// Get is a legacy method, to be removed once we have commands that don't rely on globals
func Get() *MediatorClient {
	url := api.GetServiceURL(api.ServiceMediator)
	return New(url.String(), map[string][]string{}, authentication.Get(), 0)
}

func New(url string, common Header, bearerToken BearerTokenProvider, timeout time.Duration) *MediatorClient {
	if timeout == 0 {
		timeout = time.Second * 60
	}

	retryOpt := graphql.WithHTTPClient(retryhttp.DefaultClient.StandardClient())

	return &MediatorClient{
		Client:        graphql.NewClient(url, retryOpt),
		common:        common,
		tokenProvider: bearerToken,
		timeout:       timeout,
	}
}

func (c *MediatorClient) SetDebug(b bool) {
	c.Client.Log = func(string) {}
	if b {
		c.Client.Log = func(s string) {
			fmt.Fprintln(os.Stderr, s)
		}
	}
}

func (c *MediatorClient) Run(request Request, response interface{}) error {
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

	graphRequest.Header.Set("X-Requestor", machineid.UniqID())

	return c.Client.Run(ctx, graphRequest, response)
}
