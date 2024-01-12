package vulnerabilities

import (
	"github.com/ActiveState/cli/internal/gqlclient"
	"github.com/ActiveState/cli/pkg/platform/api"
)

func New() *gqlclient.Client {
	return gqlclient.New(api.GetServiceURL(api.ServiceVulnerabilities).String(), 0)
}
