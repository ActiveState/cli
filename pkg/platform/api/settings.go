package api

import (
	"log"
	"net/url"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
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
	Schema   string
	Host     string
	BasePath string
	URL      *url.URL
}

type urlsByService map[Service]string

var UrlsByEnv = map[string]urlsByService{
	"prod": {
		ServiceMono:      constants.MonoURLProd,
		ServiceSecrets:   constants.SecretsURLProd,
		ServiceHeadChef:  constants.HeadChefURLProd,
		ServiceInventory: constants.InventoryURLProd,
	},
	"stage": {
		ServiceMono:      constants.MonoURLStage,
		ServiceSecrets:   constants.SecretsURLStage,
		ServiceHeadChef:  constants.HeadChefURLStage,
		ServiceInventory: constants.InventoryURLStage,
	},
	"dev": {
		ServiceMono:      constants.MonoURLDev,
		ServiceSecrets:   constants.SecretsURLDev,
		ServiceHeadChef:  constants.HeadChefURLDev,
		ServiceInventory: constants.InventoryURLDev,
	},
	"test": {
		ServiceMono:      "https://testing.tld" + constants.MonoAPIPath,
		ServiceSecrets:   "https://secrets.testing.tld" + constants.SecretsAPIPath,
		ServiceHeadChef:  "https://headchef.testing.tld" + constants.HeadChefAPIPath,
		ServiceInventory: "https://inventory.testing.tld" + constants.InventoryAPIPath,
	},
}

var serviceURLs = map[Service]*url.URL{}

// init determines the name of the API environment to use. It prefers a custom
// APIEnv env variable if available. If not defined or no setting found for the provided
// custom value, then the apiEnvName determines if this is test, prod, or stage based on
// a few factors. The default is always stage.
func init() {
	DetectServiceURLs()
}

func DetectServiceURLs() {
	serviceURLStrings := urlsByService{}

	if condition.InTest() {
		serviceURLStrings = UrlsByEnv["test"]
	} else {
		var hasURL bool
		if serviceURLStrings, hasURL = UrlsByEnv[constants.APIEnv]; !hasURL {
			if constants.BranchName == "prod" {
				serviceURLStrings = UrlsByEnv["prod"]
			} else {
				serviceURLStrings = UrlsByEnv["stage"]
			}
		}
	}

	for sv, urlStr := range serviceURLStrings {
		u, err := url.Parse(urlStr)
		if err != nil {
			log.Panicf("Invalid URL format: %s", urlStr)
		}
		serviceURLs[sv] = u
	}
}

// GetServiceURL returns the URL for the given service
func GetServiceURL(service Service) *url.URL {
	serviceURL, ok := serviceURLs[service]
	if !ok {
		log.Panicf("API Service does not exist: %v", service)
	}

	return serviceURL
}

// GetSettings returns the environmental settings for the specified service
func GetSettings(service Service) Settings {
	u := GetServiceURL(service)
	return Settings{u.Scheme, u.Host, u.Path, u}
}
