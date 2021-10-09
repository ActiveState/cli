package graphql

import (
	"github.com/ActiveState/cli/internal/gqlclient"
	"github.com/ActiveState/cli/pkg/platform/api"
)

func New(tokenProvider gqlclient.BearerTokenProvider) *gqlclient.Client {
	url := api.GetServiceURL(api.ServiceGraphQL)
	c := gqlclient.New(url.String(), 0)
	c.SetTokenProvider(tokenProvider)
	return c
}
