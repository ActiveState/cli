package vulnerabilities

import (
	"github.com/ActiveState/cli/internal/graphql"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

func New(auth *authentication.Auth) *graphql.Client {
	client := graphql.New(api.GetServiceURL(api.ServiceVulnerabilities).String(), 0)

	// Most requests to this service require authentication
	if auth != nil && auth.Authenticated() {
		client.SetTokenProvider(auth)
	}

	return client
}
