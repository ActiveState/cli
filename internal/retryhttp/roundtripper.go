package retryhttp

import (
	"net/http"
	"sync"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/hashicorp/go-retryablehttp"
)

// RoundTripper implements the http.RoundTripper interface, using a retrying
// HTTP client to execute requests.
type RoundTripper struct {
	Client *Client
	once   sync.Once
}

func (rt *RoundTripper) init() {
	if rt.Client == nil {
		rt.Client = DefaultClient
	}
}

// RoundTrip satisfies the http.RoundTripper interface.
func (rt *RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	retryableReq, err := retryablehttp.FromRequest(req)
	if err != nil {
		return nil, locale.WrapError(err, "err_retry_convert_req", "Could not convert request to retryable format")
	}

	return rt.Client.Do(retryableReq)
}
