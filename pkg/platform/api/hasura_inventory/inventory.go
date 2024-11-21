package inventory

import (
	"github.com/ActiveState/cli/internal/gqlclient"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

func New(auth *authentication.Auth) *gqlclient.Client {
	client := gqlclient.New(api.GetServiceURL(api.ServiceHasuraInventory).String(), 0)

	if auth != nil && auth.Authenticated() {
		client.SetTokenProvider(auth)
	}

	return client
}
