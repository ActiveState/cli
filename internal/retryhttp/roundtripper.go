package retryhttp

import (
	"net/http"
	"strings"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/api/errors"
	"github.com/hashicorp/go-retryablehttp"
)

// RoundTripper implements the http.RoundTripper interface, using a retrying
// HTTP client to execute requests.
type RoundTripper struct {
	client *Client
}

func (rt *RoundTripper) init() {
	if rt.client == nil {
		rt.client = DefaultClient
	}
}

// RoundTrip satisfies the http.RoundTripper interface.
func (rt *RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	retryableReq, err := retryablehttp.FromRequest(req)
	if err != nil {
		return nil, locale.WrapError(err, "err_retry_convert_req", "Could not convert request to retryable format")
	}

	resp, err := rt.client.Do(retryableReq)
	if err != nil && resp != nil && resp.StatusCode == http.StatusForbidden && strings.EqualFold(resp.Header.Get("server"), "cloudfront") {
		return nil, api_errors.NewCountryBlockedError()
	}

	return resp, err
}
