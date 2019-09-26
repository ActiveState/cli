package gqlclient

import (
	"context"
	"time"

	"github.com/machinebox/graphql"
)

type Header map[string][]string

type graphqlRequest = graphql.Request

type graphqlClient = graphql.Client

type GQLClient struct {
	*graphqlClient
	common  Header
	timeout time.Duration
}

func New(url string, common Header, timeout time.Duration) *GQLClient {
	if timeout == 0 {
		timeout = time.Second * 60
	}

	return &GQLClient{
		graphqlClient: graphql.NewClient(url),
		common:        common,
		timeout:       timeout,
	}
}

func (c *GQLClient) Run(req *Request, resp interface{}) error {
	ctx := req.ctx
	if ctx == nil {
		ctx = context.Background()
		var cancel context.CancelFunc

		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	return c.graphqlClient.Run(ctx, req.graphqlRequest, resp)
}

type Request struct {
	*graphqlRequest
	ctx context.Context
}

func (c *GQLClient) NewRequest(qry string) *Request {
	req := Request{
		graphqlRequest: graphql.NewRequest(qry),
	}

	for k, vs := range c.common {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}

	return &req
}

func (req *Request) Context(ctx context.Context) {
	req.ctx = ctx
}
