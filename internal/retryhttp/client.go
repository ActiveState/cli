package retryhttp

import (
	"context"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/multilog"
)

type UserNetworkError struct {
	_testCode int // used for tests
}

func (e *UserNetworkError) Error() string {
	return "network error"
}

func (e *UserNetworkError) ExitCode() int {
	return 11
}

type Logger interface {
	Printf(string, ...interface{})
}

var (
	DefaultTimeout = time.Second * 30
	DefaultRetries = 5
	DefaultClient  = NewClient(DefaultTimeout, DefaultRetries)

	// A regular expression to match the error returned by net/http when the
	// configured number of redirects is exhausted. This error isn't typed
	// specifically so we resort to matching on the error string.
	redirectsErrorRe = regexp.MustCompile(`stopped after \d+ redirects\z`)

	// A regular expression to match the error returned by net/http when the
	// scheme specified in the URL is invalid. This error isn't typed
	// specifically so we resort to matching on the error string.
	schemeErrorRe = regexp.MustCompile(`unsupported protocol scheme`)

	retryableStatusCodes = []int{
		// 4XX Status codes

		// The server timed out waiting for the request from client.
		http.StatusRequestTimeout,
		// Sometimes the server puts a Retry-After response header
		// to indicate when the server is available to start processing
		// request from client.
		http.StatusTooManyRequests,
		// The server is unwilling to risk
		// processing a request that might be replayed.
		http.StatusTooEarly,
		// 5XX Status codes

		// The server, while acting as a gateway or proxy, did not receive
		// a valid response.
		http.StatusBadGateway,
		// The server is currently unable to handle the request due to a
		// temporary overload or scheduled maintenance, which will likely
		// be alleviated after some delay.
		http.StatusServiceUnavailable,
		// The server, while acting as a gateway or proxy, did not receive
		// a timely response from an upstream server it needed to access
		// in order to complete the request.
		http.StatusGatewayTimeout,
	}
)

type Client struct {
	*retryablehttp.Client
}

func (c *Client) Get(url string) (*http.Response, error) {
	return normalizeResponse(c.Client.Get(url))
}

func (c *Client) Head(url string) (*http.Response, error) {
	return normalizeResponse(c.Client.Head(url))
}

func (c *Client) Post(url, bodyType string, body interface{}) (*http.Response, error) {
	return normalizeResponse(c.Client.Post(url, bodyType, body))
}

func (c *Client) PostForm(url string, data url.Values) (*http.Response, error) {
	return normalizeResponse(c.Client.PostForm(url, data))
}

func (c *Client) Do(req *retryablehttp.Request) (*http.Response, error) {
	return normalizeResponse(c.Client.Do(req))
}

func (c *Client) StandardClient() *http.Client {
	return &http.Client{
		Transport: &RoundTripper{client: c},
	}
}

func normalizeResponse(res *http.Response, err error) (*http.Response, error) {
	if res != nil {
		switch res.StatusCode {
		case 408:
			return res, locale.WrapExternalError(&UserNetworkError{408}, "err_user_network_server_timeout", "Request failed due to timeout during communication with server. {{.V0}}", locale.Tr("err_user_network_solution", constants.ForumsURL))
		case 425:
			return res, locale.WrapExternalError(&UserNetworkError{425}, "err_user_network_tooearly", "Request failed due to retrying connection too fast. {{.V0}}", locale.Tr("err_user_network_solution", constants.ForumsURL))
		case 429:
			return res, locale.WrapExternalError(&UserNetworkError{429}, "err_user_network_toomany", "Request failed due to too many requests. {{.V0}}", locale.Tr("err_user_network_solution", constants.ForumsURL))
		}
	}

	var dnsError *net.DNSError
	if errors.As(err, &dnsError) {
		return res, locale.WrapExternalError(&UserNetworkError{}, "err_user_network_dns", "Request failed due to DNS error: {{.V0}}. {{.V1}}", err.Error(), locale.Tr("err_user_network_solution", constants.ForumsURL))
	}

	// Due to Go's handling of these types of errors and due to Windows localizing the errors in question we have to rely on the `wsarecv:` keyword to capture a series
	// of user facing network issues. Theoretically this could cause some false positives, but at the time of writing I could not find any instances on rollbar
	// where `wsarecv:` was being reported as anything other than a network issue caused by the user or their network
	if err != nil && strings.Contains(err.Error(), "wsarecv:") {
		multilog.Error("Non-Critical User Network Issue, please vet for false-positive: %v", err) // Logging so we can vet for false positives
		return res, locale.WrapError(&UserNetworkError{}, "err_user_network_wsarecv", "Request failed due to user network error: {{.V0}}. {{.V1}}", err.Error(), locale.Tr("err_user_network_solution", constants.ForumsURL))
	}

	return res, err
}

func normalizeRetryResponse(res *http.Response, err error, numTries int) (*http.Response, error) {
	logging.Debug("Retry failed with error: %v, after %d tries", err, numTries)
	if err2, ok := err.(net.Error); ok && err2.Timeout() {
		return res, locale.WrapExternalError(&UserNetworkError{-1}, "err_user_network_timeout", "", locale.Tr("err_user_network_solution", constants.ForumsURL))
	}
	return res, err
}

func NewClient(timeout time.Duration, retries int) *Client {
	if timeout < 0 {
		timeout = DefaultTimeout
	}
	if retries < 0 {
		retries = DefaultRetries
	}

	retryClient := retryablehttp.NewClient()
	retryClient.Logger = nil
	// retryClient.Logger = logging.CurrentHandler() // Enable this to get debug info in our logs
	retryClient.HTTPClient = &http.Client{
		Transport: transport(),
		Timeout:   timeout,
	}
	retryClient.RetryMax = retries
	retryClient.ErrorHandler = normalizeRetryResponse
	retryClient.CheckRetry = retryPolicy

	return &Client{
		Client: retryClient,
	}
}

func transport() http.RoundTripper {
	if condition.InUnitTest() {
		return http.DefaultTransport
	}
	return cleanhttp.DefaultPooledTransport()
}

// retryPolicy is a modified version of retryablehttp.DefaultRetryPolicy to handle
// status codes differently.
func retryPolicy(ctx context.Context, resp *http.Response, err error) (bool, error) {
	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	if err != nil {
		if v, ok := err.(*url.Error); ok {
			// Don't retry if the error was due to too many redirects.
			if redirectsErrorRe.MatchString(v.Error()) {
				return false, nil
			}

			// Don't retry if the error was due to an invalid protocol scheme.
			if schemeErrorRe.MatchString(v.Error()) {
				return false, nil
			}

			// Don't retry if the error was due to TLS cert verification failure.
			if _, ok := v.Err.(x509.UnknownAuthorityError); ok {
				return false, nil
			}
		}

		// The error is likely recoverable so retry.
		return true, err
	}

	return isRetryableStatus(resp.StatusCode), nil
}

func isRetryableStatus(status int) bool {
	return funk.Contains(retryableStatusCodes, status)
}
