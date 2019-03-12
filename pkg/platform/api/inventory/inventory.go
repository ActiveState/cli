package inventory

import (
	"flag"
	"net/http"

	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_client"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_client/inventory_operations"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
)

// persist contains the active API Client connection
var persist *inventory_operations.Client

var transport http.RoundTripper

// Init will create a new API client using default settings
func Init() *inventory_operations.Client {
	return New(api.GetSettings(api.ServiceInventory))
}

// New initializes a new api client
func New(apiSetting api.Settings) *inventory_operations.Client {
	transportRuntime := httptransport.New(apiSetting.Host, apiSetting.BasePath, []string{apiSetting.Schema})
	transportRuntime.Transport = api.NewUserAgentTripper()

	if flag.Lookup("test.v") != nil {
		transportRuntime.SetDebug(true)
	}

	return inventory_client.New(transportRuntime, strfmt.Default).InventoryOperations
}

// Get returns a cached version of the default api client
func Get() *inventory_operations.Client {
	if persist == nil {
		persist = Init()
	}
	return persist
}
