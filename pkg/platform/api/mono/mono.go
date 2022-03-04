package mono

import (
	"context"
	"net"
	"net/http"
	"net/url"

	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client"
)

// persist contains the active API Client connection
var persist *mono_client.Mono

// New will create a new API client using default settings (for an authenticated version use the NewWithAuth version)
func New() *mono_client.Mono {
	return Init(api.GetServiceURL(api.ServiceMono), nil)
}

// NewWithAuth creates a new API client using default settings and the provided authentication info
func NewWithAuth(auth *runtime.ClientAuthInfoWriter) *mono_client.Mono {
	return Init(api.GetServiceURL(api.ServiceMono), auth)
}

// Init initializes a new api client
func Init(serviceURL *url.URL, auth *runtime.ClientAuthInfoWriter) *mono_client.Mono {
	transportRuntime := httptransport.New(serviceURL.Host, serviceURL.Path, []string{serviceURL.Scheme})
	transportRuntime.Transport = api.NewRoundTripper()

	// transportRuntime.SetDebug(true)

	if auth != nil {
		transportRuntime.DefaultAuthentication = *auth
	}
	client := mono_client.New(transportRuntime, strfmt.Default)

	// For the Oauth client, prefer use of IPv4. This is needed particularly for device
	// authorization, which compares the IP address of the State Tool request and the IP address of
	// the web client request. If there's a mismatch, authorization fails. When this happens, it's
	// often because the State Tool connects to the Platform via IPv6, but the browser does via IPv4
	// (browsers apparently prefer IPv4 for now).
	ipv4PreferredTransportRuntime := httptransport.New(serviceURL.Host, serviceURL.Path, []string{serviceURL.Scheme})
	ipv4PreferredTransport := http.DefaultTransport.(*http.Transport).Clone()
	ipv4PreferredTransport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		dialer := &net.Dialer{}
		if conn, err := dialer.DialContext(ctx, "tcp4", addr); conn != nil {
			return conn, err
		}
		return dialer.DialContext(ctx, "tcp", addr) // fallback to default ipv6/ipv4 dialer
	}
	ipv4PreferredTransportRuntime.Transport = api.NewRoundTripperWithTransport(ipv4PreferredTransport)
	client.Oauth.SetTransport(ipv4PreferredTransportRuntime)

	return client
}

// Get returns a cached version of the default api client
func Get() *mono_client.Mono {
	if persist == nil {
		persist = New()
	}
	return persist
}
