package loghttp

import (
	"fmt"
	"net/http"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/hashicorp/go-cleanhttp"
)

// LogFunc defines the behavior to be called to output HTTP client request logs.
type LogFunc func(...interface{})

// Transport manages the context required to log HTTP client requests.
type Transport struct {
	Transport http.RoundTripper
	LogFn     LogFunc
}

// NewTransport returns a pointer to a prepared instance of Transport.
func NewTransport(fn LogFunc) *Transport {
	return &Transport{
		Transport: transport(),
		LogFn:     fn,
	}
}

// RoundTrip implements http.RoundTripper.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.LogFn(fmt.Sprintf("HTTP Client [%s] %s", req.Method, req.URL))

	return t.Transport.RoundTrip(req)
}

func transport() http.RoundTripper {
	if condition.InTest() {
		return http.DefaultClient.Transport
	}
	return cleanhttp.DefaultPooledTransport()
}
