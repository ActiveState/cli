package graphql

import (
	"github.com/ActiveState/cli/internal/gqlclient"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

func New(auth *authentication.Auth, services ...api.Service) *gqlclient.Client {
	var service api.Service = api.ServiceGraphQL
	if len(services) > 0 {
		service = services[0]
	}

	url := api.GetServiceURL(service)
	c := gqlclient.New(url.String(), 0)
	c.SetTokenProvider(auth)
	return c
}
