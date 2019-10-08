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

// Settings encapsulates settings needed for an API endpoint
type Settings struct {
	Scheme   string
	Host     string
	BasePath string
}

var urlsByService = map[Service]Settings{
	ServiceMono: {
		Scheme:   "https",
		Host:     constants.DefaultAPIHost,
		BasePath: constants.MonoAPIPath,
	},
	ServiceSecrets: {
		Scheme:   "https",
		Host:     constants.DefaultAPIHost,
		BasePath: constants.SecretsAPIPath,
	},
	ServiceHeadChef: {
		Scheme:   "wss",
		Host:     constants.DefaultAPIHost,
		BasePath: constants.HeadChefAPIPath,
	},
	ServiceInventory: {
		Scheme:   "https",
		Host:     constants.DefaultAPIHost,
		BasePath: constants.InventoryAPIPath,
	},
}

// GetServiceURL returns the URL for the given service
func GetServiceURL(service Service) *url.URL {
	settings := GetSettings(service)
	return &url.URL{
		Scheme: settings.Scheme,
		Host:   settings.Host,
		Path:   settings.BasePath,
	}
}

// GetSettings returns the environmental settings for the specified service
func GetSettings(service Service) Settings {
	settings := urlsByService[service]
	if condition.InTest() {
		settings.Host = string(service) + ".testing.tld"
	} else if host := getProjectHost(); host != nil {
		settings.Host = *host
	}
	return settings
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
