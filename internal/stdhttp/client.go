package stdhttp

import (
	"net/http"
	"time"

	"github.com/hashicorp/go-cleanhttp"
)

var (
	// DefaultPooledTransport is the application-wide pooled HTTP transport.
	DefaultPooledTransport = cleanhttp.DefaultPooledTransport()

	// DefaultClient is the application-wide default HTTP client.
	DefaultClient = NewClient(time.Second * 30)
)

// NewClient returns an instance of http.Client using a clean transport
func NewClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Transport: DefaultPooledTransport,
		Timeout:   timeout,
	}
}
