package api

import (
	"net/url"
	"os"
	"strings"

	"github.com/ActiveState/cli/pkg/projectfile"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/logging"
)

// Service records available api services
type Service string

const (
	// ServiceMono is our main service for api services, "Mono" refers to its monolithic nature, one that we're trying to get away from
	ServiceMono Service = "platform"

	// ServiceSecrets is our service that's used purely for setting and storing secrets
	ServiceSecrets = "secrets"

	// ServiceHeadChef is our service that's used to kick off and track builds
	ServiceHeadChef = "headchef"

	// ServiceHeadChefWS is the websocket service on headchef
	BuildLogStreamer = "buildlog-streamer"

	// ServiceInventory is our service that's used to query available inventory and dependencies
	ServiceInventory = "inventory"

	// ServiceGraphQL is our service that's used as a graphql endpoint for platform requests
	ServiceGraphQL = "platform-graphql"

	// ServiceMediator is our mediator service used to query build graph data
	ServiceMediator = "mediator"

	// ServiceRequirementsImport is our service that processes requirements.txt files.
	ServiceRequirementsImport = "requirements-import"
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
		Scheme: "https",
		Host:   constants.DefaultAPIHost,
		Path:   constants.HeadChefAPIPath,
	},
	BuildLogStreamer: {
		Scheme: "wss",
		Host:   constants.DefaultAPIHost,
		Path:   constants.BuildLogStreamerPath,
	},
	ServiceInventory: {
		Scheme: "https",
		Host:   constants.DefaultAPIHost,
		Path:   constants.InventoryAPIPath,
	},
	ServiceGraphQL: {
		Scheme: "https",
		Host:   constants.DefaultAPIHost,
		Path:   constants.GraphqlAPIPath,
	},
	ServiceMediator: {
		Scheme: "https",
		Host:   constants.DefaultAPIHost,
		Path:   constants.MediatorAPIPath,
	},
	ServiceRequirementsImport: {
		Scheme: "https",
		Host:   constants.DefaultAPIHost,
		Path:   constants.RequirementsImportAPIPath,
	},
}

// GetServiceURL returns the URL for the given service
func GetServiceURL(service Service) *url.URL {
	serviceURL, validService := urlsByService[service]
	if !validService {
		logging.Panic("Invalid service: %s", string(service))
	}
	if host := getProjectHost(service); host != nil {
		serviceURL.Host = *host
	}

	if insecure := os.Getenv(constants.APIInsecureEnvVarName); insecure == "true" {
		if serviceURL.Scheme == "https" || serviceURL.Scheme == "wss" {
			serviceURL.Scheme = strings.TrimRight(serviceURL.Scheme, "s")
		}
	}

	return serviceURL
}

func getProjectHost(service Service) *string {
	if apiHost := os.Getenv(constants.APIHostEnvVarName); apiHost != "" {
		return &apiHost
	}

	if condition.InTest() {
		testingPlatform := string(service) + ".testing.tld"
		return &testingPlatform
	}

	pj, err := projectfile.GetOnce()
	if err != nil {
		return nil
	}
	url, err := url.Parse(pj.Project)
	if err != nil {
		logging.Error("Could not parse project url: %s", pj.Project)
		return nil
	}

	return &url.Host
}
