package api

import (
	"github.com/ActiveState/cli/internal/api/client"
	"github.com/ActiveState/cli/internal/constants"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
)

var Client *client.APIClient

func init() {
	transport := httptransport.New(constants.ApiHost, constants.ApiPath, []string{constants.ApiSchema})
	Client = client.New(transport, strfmt.Default)
}
