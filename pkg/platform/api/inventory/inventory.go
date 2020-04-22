package inventory

import (
	"net/url"

	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_client"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_client/inventory_operations"
)

// persist contains the active API Client connection
var persist *inventory_operations.Client

// Init will create a new API client using default settings
func Init() *inventory_operations.Client {
	return New(api.GetServiceURL(api.ServiceInventory))
}

// New initializes a new api client
func New(serviceURL *url.URL) *inventory_operations.Client {
	transportRuntime := httptransport.New(serviceURL.Host, serviceURL.Path, []string{serviceURL.Scheme})
	transportRuntime.Transport = api.NewUserAgentTripper()

	//transportRuntime.SetDebug(true)

	return inventory_client.New(transportRuntime, strfmt.Default).InventoryOperations
}

// Get returns a cached version of the default api client
func Get() *inventory_operations.Client {
	if persist == nil {
		persist = Init()
	}
	return persist
}
