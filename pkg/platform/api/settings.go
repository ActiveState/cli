package api

import (
	"net/url"
	"os"
	"strings"

	configMediator "github.com/ActiveState/cli/internal/mediators/config"
	"github.com/ActiveState/cli/pkg/projectfile"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/logging"
)

func init() {
	configMediator.RegisterOption(constants.APIHostConfig, configMediator.String, "")
}

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

	// ServiceBuildPlanner is our service that processes build plans.
	ServiceBuildPlanner = "build-planner"

	// ServiceVulnerabilities is Data Acquisition's Hasura service for vulnerability (CVE) information.
	ServiceVulnerabilities = "vulnerabilities"

	// ServiceHasuraInventory is the Hasura service for inventory information.
	ServiceHasuraInventory = "hasura-inventory"

	// ServiceUpdateInfo is the service for update info
	ServiceUpdateInfo = "update-info"

	// TestingPlatform is the API host used by tests so-as not to affect production.
	TestingPlatform = ".testing.tld"
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
	ServiceBuildPlanner: {
		Scheme: "https",
		Host:   constants.DefaultAPIHost,
		Path:   constants.BuildPlannerAPIPath,
	},
	ServiceVulnerabilities: {
		Scheme: "https",
		Host:   constants.DefaultAPIHost,
		Path:   constants.VulnerabilitiesAPIPath,
	},
	ServiceHasuraInventory: {
		Scheme: "https",
		Host:   constants.DefaultAPIHost,
		Path:   constants.HasuraInventoryAPIPath,
	},
	ServiceUpdateInfo: {
		Scheme: "https",
		Host:   constants.DefaultAPIHost,
		Path:   constants.UpdateInfoAPIPath,
	},
}

var currentCfg *config.Instance

func SetConfig(config *config.Instance) {
	currentCfg = config
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

	if insecure := os.Getenv(constants.APIHostEnvVarName); insecure == "true" {
		if serviceURL.Scheme == "https" || serviceURL.Scheme == "wss" {
			serviceURL.Scheme = strings.TrimRight(serviceURL.Scheme, "s")
		}
	}

	sname := strings.Replace(strings.ToUpper(string(service)), "-", "_", -1)
	envname := constants.APIServiceOverrideEnvVarName + sname
	if override := os.Getenv(envname); override != "" {
		u, err := url.Parse(override)
		if err != nil {
			logging.Error("Could not apply %s: %s", envname, err)
		} else {
			return u
		}
	}

	return serviceURL
}

func getProjectHost(service Service) *string {
	if host := HostOverride(); host != "" {
		logging.Debug("Using host override: %s", host)
		return &host
	}

	if condition.InUnitTest() {
		testingPlatform := string(service) + TestingPlatform
		return &testingPlatform
	}

	pj, err := projectfile.FromEnv()
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

func getProjectHostFromConfig() string {
	if currentCfg != nil && !currentCfg.Closed() {
		return currentCfg.GetString(constants.APIHostConfig)
	}
	return ""
}

func HostOverride() string {
	if apiHost := os.Getenv(constants.APIHostEnvVarName); apiHost != "" {
		return apiHost
	}

	if apiHost := getProjectHostFromConfig(); apiHost != "" {
		return apiHost
	}

	return ""
}

// GetPlatformURL returns a generic Platform URL for the given path.
// This is for retrieving non-service URLs (e.g. signup URL).
func GetPlatformURL(path string) *url.URL {
	host := constants.DefaultAPIHost
	if hostOverride := HostOverride(); hostOverride != "" {
		host = hostOverride
	}
	return &url.URL{
		Scheme: "https",
		Host:   host,
		Path:   path,
	}
}
