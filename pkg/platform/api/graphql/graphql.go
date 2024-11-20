package graphql

import (
	"github.com/ActiveState/cli/internal/graphql"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

func New(auth *authentication.Auth) *graphql.Client {
	url := api.GetServiceURL(api.ServiceGraphQL)
	c := graphql.New(url.String(), 0)
	c.SetTokenProvider(auth)
	return c
}
