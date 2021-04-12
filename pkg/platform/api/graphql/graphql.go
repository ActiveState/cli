package graphql

import (
	"github.com/ActiveState/cli/internal/gqlclient"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

func New() *gqlclient.Client {
	url := api.GetServiceURL(api.ServiceGraphQL)
	c := gqlclient.New(url.String(), 0)
	c.SetTokenProvider(authentication.Get())
	return c
}
