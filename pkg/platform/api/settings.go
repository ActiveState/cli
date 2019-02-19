package api

import (
	"flag"
	"log"
	"net/url"

	"github.com/ActiveState/cli/internal/constants"
)

// Service records available api serices
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

var serviceURLs urlsByService

// init determines the name of the API environment to use. It prefers a custom
// APIEnv env variable if available. If not defined or no setting found for the provided
// custom value, then the apiEnvName determines if this is test, prod, or stage based on
// a few factors. The default is always stage.
func init() {
	var hasURL bool
	if serviceURLs, hasURL = urlsByEnv[constants.APIEnv]; !hasURL {
		if flag.Lookup("test.v") != nil {
			serviceURLs = urlsByEnv["test"]
		} else if constants.BranchName == "prod" {
			serviceURLs = urlsByEnv["prod"]
		} else {
			serviceURLs = urlsByEnv["stage"]
		}
	}
}

// GetServiceURL returns the URL for the given service
func GetServiceURL(service Service) *url.URL {
	serviceURL, ok := serviceURLs[service]
	if !ok {
		log.Panicf("API Service does not exist: %v", service)
	}

	u, err := url.Parse(serviceURL)
	if err != nil {
		log.Panicf("Invalid URL format: %s", serviceURL)
	}

	return u
}

// GetSettings returns the environmental settings for the specified service
func GetSettings(service Service) Settings {
	u := GetServiceURL(service)
	return Settings{u.Scheme, u.Host, u.Path}
}
