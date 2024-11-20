package inventory

import (
	"github.com/ActiveState/cli/internal/graphql"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

func New(auth *authentication.Auth) *graphql.Client {
	client := graphql.New(api.GetServiceURL(api.ServiceHasuraInventory).String(), 0)

	if auth != nil && auth.Authenticated() {
		client.SetTokenProvider(auth)
	}

	return client
}
