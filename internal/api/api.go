package api

import (
	"github.com/ActiveState/cli/internal/api/client"
	"github.com/ActiveState/cli/internal/constants"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
)

// Client contains the active API Client connection
var Client *client.APIClient

func init() {
	transport := httptransport.New(constants.APIHost, constants.APIPath, []string{constants.APISchema})
	Client = client.New(transport, strfmt.Default)
}
