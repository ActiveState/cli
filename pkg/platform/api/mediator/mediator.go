package mediator

import (
	"github.com/ActiveState/cli/internal/gqlclient"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

func New(auth *authentication.Auth) *gqlclient.Client {
	url := api.GetServiceURL(api.ServiceMediator)
	c := gqlclient.New(url.String(), 0)
	c.SetTokenProvider(auth)
	return c
}
