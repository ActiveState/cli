package retryhttp

import (
	"net/http"

	"github.com/hashicorp/go-retryablehttp"
)

var DefaultClient = NewClient()

type Client struct {
	*retryablehttp.Client
}

func NewClient() *Client {
	retryClient := retryablehttp.NewClient()
	retryClient.Logger = nil                    // silence debugging
	retryClient.HTTPClient = http.DefaultClient // use default: httpmock registers w/default

	return &Client{
		Client: retryClient,
	}
}
