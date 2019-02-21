package api

import (
	"flag"
	"log"
	"net/url"

	"github.com/ActiveState/cli/internal/constants"
)

// Service records available api services
type Service string

const (
	// ServicePlatform is our main service for api endpoints
	ServicePlatform Service = "platform"

	// ServiceSecrets is our service that's used purely for setting and storing secrets
	ServiceSecrets = "secrets"
)

// Settings encapsulates settings needed for an API endpoint
type Settings struct {
	Schema   string
	Host     string
	BasePath string
}

type urlsByService map[Service]string

var urlsByEnv = map[string]urlsByService{
	"prod": {
		ServicePlatform: constants.PlatformURLProd,
		ServiceSecrets:  constants.SecretsURLProd,
	},
	"stage": {
		ServicePlatform: constants.PlatformURLStage,
		ServiceSecrets:  constants.SecretsURLStage,
	},
	"dev": {
		ServicePlatform: constants.PlatformURLDev,
		ServiceSecrets:  constants.SecretsURLDev,
	},
	"test": {
		ServicePlatform: "https://testing.tld" + constants.PlatformAPIPath,
		ServiceSecrets:  "https://secrets.testing.tld" + constants.SecretsAPIPath,
	},
}

var serviceURLs = map[Service]*url.URL{}

// init determines the name of the API environment to use. It prefers a custom
// APIEnv env variable if available. If not defined or no setting found for the provided
// custom value, then the apiEnvName determines if this is test, prod, or stage based on
// a few factors. The default is always stage.
func init() {
	serviceURLStrings := urlsByService{}

	var hasURL bool
	if serviceURLStrings, hasURL = urlsByEnv[constants.APIEnv]; !hasURL {
		if flag.Lookup("test.v") != nil {
			serviceURLStrings = urlsByEnv["test"]
		} else if constants.BranchName == "prod" {
			serviceURLStrings = urlsByEnv["prod"]
		} else {
			serviceURLStrings = urlsByEnv["stage"]
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
	return Settings{u.Scheme, u.Host, u.Path}
}
