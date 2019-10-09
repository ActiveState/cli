package api

import (
	"net/url"

	"github.com/ActiveState/cli/pkg/projectfile"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/logging"
)

// Service records available api services
type Service string

const (
	// ServiceMono is our main service for api endpoints, "Mono" refers to its monolithic nature, one that we're trying to get away from
	ServiceMono Service = "platform"

	// ServiceSecrets is our service that's used purely for setting and storing secrets
	ServiceSecrets = "secrets"

	// ServiceHeadChef is our service that's used to kick off and track builds
	ServiceHeadChef = "headchef"

	// ServiceInventory is our service that's used to query available inventory and dependencies
	ServiceInventory = "inventory"
)

var urlsByService = map[Service]*url.URL{
	ServiceMono: {
		Scheme: "https",
		Host:   constants.DefaultAPIHost,
		Path:   constants.MonoAPIPath,
	},
	ServiceSecrets: {
		Scheme: "https",
		Host:   constants.DefaultAPIHost,
		Path:   constants.SecretsAPIPath,
	},
	ServiceHeadChef: {
		Scheme: "wss",
		Host:   constants.DefaultAPIHost,
		Path:   constants.HeadChefAPIPath,
	},
	ServiceInventory: {
		Scheme: "https",
		Host:   constants.DefaultAPIHost,
		Path:   constants.InventoryAPIPath,
	},
}

// GetServiceURL returns the URL for the given service
func GetServiceURL(service Service) *url.URL {
	serviceURL, validService := urlsByService[service]
	if !validService {
		logging.Panic("Invalid service: %s", string(service))
	}
	if condition.InTest() {
		serviceURL.Host = string(service) + ".testing.tld"
	} else if host := getProjectHost(); host != nil {
		serviceURL.Host = *host
	}
	return serviceURL
}

func getProjectHost() *string {
	pj, fail := projectfile.GetOnce()
	if fail != nil {
		return nil
	}
	url, err := url.Parse(pj.Project)
	if err != nil {
		logging.Error("Could not parse project url: %s", pj.Project)
		return nil
	}

	return &url.Host
}
