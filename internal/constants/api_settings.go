package constants

import (
	"flag"
)

const (
	platformKey = "platform"
	secretsKey  = "secrets"
)

// APISetting encapsulates settings needed for an API endpoint
type APISetting struct {
	Schema   string
	Host     string
	BasePath string
}

var envSettings = map[string]map[string]APISetting{
	"prod": {
		platformKey: APISetting{"https", "platform.activestate.com", "/api/v1"},
		secretsKey:  APISetting{"https", "platform.activestate.com", "/api/secrets/v1"},
	},
	"stage": {
		platformKey: APISetting{"https", "staging.activestate.build", "/api/v1"},
		secretsKey:  APISetting{"https", "staging.activestate.build", "/api/secrets/v1"},
	},
	"dev": {
		platformKey: APISetting{"https", "staging.activestate.build", "/api/v1"},
		secretsKey:  APISetting{"http", "localhost:8080", "/api/secrets/v1"},
	},
	"test": {
		platformKey: APISetting{"https", "testing.tld", "/api/v1"},
		secretsKey:  APISetting{"https", "secrets.testing.tld", "/api/secrets/v1"},
	},
}

var envName string

// getEnvName memoizes the name of the env to use. It prefers a custom env if available
// and if one is not found, then determines if this is test, prod, or stage. Defaults to
// stage.
func getEnvName() string {
	if envName == "" {
		envName = EnvName
		if _, hasSettingsForEnv := envSettings[envName]; !hasSettingsForEnv {
			if flag.Lookup("test.v") != nil {
				envName = "test"
			} else if BranchName == "prod" {
				envName = "prod"
			} else {
				envName = "stage"
			}
		}
	}
	return envName
}

// GetPlatformAPISettings returns the environmental settings for the platform api
func GetPlatformAPISettings() APISetting {
	return envSettings[getEnvName()][platformKey]
}

// GetSecretsAPISettings returns the environmental settings for the secrets api
func GetSecretsAPISettings() APISetting {
	return envSettings[getEnvName()][secretsKey]
}
