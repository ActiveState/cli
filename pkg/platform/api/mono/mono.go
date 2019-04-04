package mono

import (
	"os"

	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client"
	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/thoas/go-funk"
)

// persist contains the active API Client connection
var persist *mono_client.Mono

// New will create a new API client using default settings (for an authenticated version use the NewWithAuth version)
func New() *mono_client.Mono {
	return Init(api.GetSettings(api.ServiceMono), nil)
}

// NewWithAuth creates a new API client using default settings and the provided authentication info
func NewWithAuth(auth *runtime.ClientAuthInfoWriter) *mono_client.Mono {
	return Init(api.GetSettings(api.ServiceMono), auth)
}

// Init initializes a new api client
func Init(apiSetting api.Settings, auth *runtime.ClientAuthInfoWriter) *mono_client.Mono {
	transportRuntime := httptransport.New(apiSetting.Host, apiSetting.BasePath, []string{apiSetting.Schema})
	transportRuntime.Transport = api.NewUserAgentTripper()

	if funk.Contains(os.Args, "-v") {
		transportRuntime.SetDebug(true)
	}

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
