package svc

import (
	"fmt"
	"net/http"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/gqlclient"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/machinebox/graphql"
)

type Client struct {
	*gqlclient.Client
	baseUrl string
}

type configurable interface {
	GetInt(string) int
}

// New will create a new API client using default settings (for an authenticated version use the NewWithAuth version)
func New(cfg configurable) (*Client, error) {
	port := cfg.GetInt(constants.SvcConfigPort)
	if port <= 0 {
		return nil, locale.NewError("err_svc_no_port", "The State Tool service does not appear to be running (no local port was configured).")
	}

	baseUrl := fmt.Sprintf("http://127.0.0.1:%d", port)
	return &Client{
		// The custom client bypasses http-retry, which we don't need for doing local requests
		Client:  gqlclient.NewWithOpts(fmt.Sprintf("%s/query", baseUrl), 0, graphql.WithHTTPClient(&http.Client{})),
		baseUrl: baseUrl,
	}, nil
}

func (c *Client) BaseUrl() string {
	return c.baseUrl
}
