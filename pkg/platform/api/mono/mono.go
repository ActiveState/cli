package mono

import (
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
	return mono_client.New(transportRuntime, strfmt.Default)
}

// Get returns a cached version of the default api client
func Get() *mono_client.Mono {
	if persist == nil {
		persist = New()
	}
	return persist
}
