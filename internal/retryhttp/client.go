package retryhttp

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/go-retryablehttp"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
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

var solutionLocale = "Internet access required"

func init() {
	solutionLocale = locale.Tl("err_user_network_solution",
		`Please ensure your device has access to internet during installation. Make sure software like Firewalls or Anti-Virus are not blocking your connectivity.`+
			`If your issue persists consider reporting it on our forums at {{.V0}}.`, constants.ForumsURL)
}

type Logger interface {
	Printf(string, ...interface{})
}

var (
	DefaultTimeout = time.Second * 30
	DefaultRetries = 5
	DefaultClient  = NewClient(DefaultTimeout, DefaultRetries)
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
			return res, locale.WrapInputError(&UserNetworkError{408}, "err_user_network_server_timeout", "Request failed due to timeout during communication with server. {{.V0}}", solutionLocale)
		case 425:
			return res, locale.WrapInputError(&UserNetworkError{425}, "err_user_network_tooearly", "Request failed due to retrying connection too fast. {{.V0}}", solutionLocale)
		case 429:
			return res, locale.WrapInputError(&UserNetworkError{429}, "err_user_network_toomany", "Request failed due to too many requests. {{.V0}}", solutionLocale)
		}
	}

	var dnsError *net.DNSError
	if errors.As(err, &dnsError) {
		return res, locale.WrapError(&UserNetworkError{}, "err_user_network_dns", "Request failed due to DNS error: {{.V0}}. {{.V1}}", err.Error(), solutionLocale)
	}

	// Due to Go's handling of these types of errors and due to Windows localizing the errors in question we have to rely on the `wsarecv:` keyword to capture a series
	// of user facing network issues. Theoretically this could cause some false positives, but at the time of writing I could not find any instances on rollbar
	// where `wsarecv:` was being reported as anything other than a network issue caused by the user or their network
	if err != nil && strings.Contains(err.Error(), "wsarecv:") {
		logging.Error("Non-Critical User Network Issue, please vet for false-positive: %v", err) // Logging so we can vet for false positives
		return res, locale.WrapError(&UserNetworkError{}, "err_user_network_wsarecv", "Request failed due to user network error: {{.V0}}. {{.V1}}", err.Error(), solutionLocale)
	}

	return res, err
}

func normalizeRetryResponse(res *http.Response, err error, numTries int) (*http.Response, error) {
	if err2, ok := err.(net.Error); ok && err2.Timeout() {
		return res, locale.WrapInputError(&UserNetworkError{-1}, "err_user_network_timeout", "Request failed due to timeout. {{.V0}}", solutionLocale)
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
	retryClient.Logger = logging.CurrentHandler()
	retryClient.HTTPClient = &http.Client{
		Transport: transport(),
		Timeout:   timeout,
	}
	retryClient.RetryMax = retries
	retryClient.ErrorHandler = normalizeRetryResponse

	return &Client{
		Client: retryClient,
	}
}

func transport() http.RoundTripper {
	if condition.InTest() {
		return http.DefaultClient.Transport
	}
	return cleanhttp.DefaultPooledTransport()
}
