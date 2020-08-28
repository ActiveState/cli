package retryhttp

import (
	"net/http"

	"github.com/ActiveState/cli/internal/logging"
	"github.com/hashicorp/go-retryablehttp"
)

type Logger interface {
	Printf(string, ...interface{})
}

var DefaultClient = NewClient(logging.CurrentHandler(), http.DefaultClient)

type Client struct {
	*retryablehttp.Client
}

func NewClient(l Logger, cl *http.Client) *Client {
	retryClient := retryablehttp.NewClient()
	retryClient.Logger = l
	retryClient.HTTPClient = cl

	return &Client{
		Client: retryClient,
	}
}
