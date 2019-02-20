package inventory

import (
	"flag"
	"net/http"

	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_client"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_client/inventory_operations"
	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
)

// persist contains the active API Client connection
var persist *inventory_operations.Client

var transport http.RoundTripper

// New will create a new API client using default settings (for an authenticated version use the NewWithAuth version)
func New() *inventory_operations.Client {
	return Init(api.GetSettings(api.ServiceInventory), nil)
}

// Init initializes a new api client
func Init(apiSetting api.Settings, auth *runtime.ClientAuthInfoWriter) *inventory_operations.Client {
	transportRuntime := httptransport.New(apiSetting.Host, apiSetting.BasePath, []string{apiSetting.Schema})
	if flag.Lookup("test.v") != nil {
		transportRuntime.SetDebug(true)
		transportRuntime.Transport = transport
	}

	if auth != nil {
		transportRuntime.DefaultAuthentication = *auth
	}
	return inventory_client.New(transportRuntime, strfmt.Default).InventoryOperations
}

// Get returns a cached version of the default api client
func Get() *inventory_operations.Client {
	if persist == nil {
		persist = New()
	}
	return persist
}
