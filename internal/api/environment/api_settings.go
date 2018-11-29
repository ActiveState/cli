package environment

import (
	"flag"

	"github.com/ActiveState/cli/internal/constants"
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

type apiSettingMap map[string]APISetting

var apiEnvSettings = map[string]apiSettingMap{
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

var apiEnvSetting apiSettingMap

// init determines the name of the API environment to use. It prefers a custom
// APIEnv env variable if available. If not defined or no setting found for the provided
// custom value, then the apiEnvName determines if this is test, prod, or stage based on
// a few factors. The default is always stage.
func init() {
	var hasSettingsForEnv bool
	if apiEnvSetting, hasSettingsForEnv = apiEnvSettings[constants.APIEnv]; !hasSettingsForEnv {
		if flag.Lookup("test.v") != nil {
			apiEnvSetting = apiEnvSettings["test"]
		} else if constants.BranchName == "prod" {
			apiEnvSetting = apiEnvSettings["prod"]
		} else {
			apiEnvSetting = apiEnvSettings["stage"]
		}
	}
}

// GetPlatformAPISettings returns the environmental settings for the platform api
func GetPlatformAPISettings() APISetting {
	return apiEnvSetting[platformKey]
}

// GetSecretsAPISettings returns the environmental settings for the secrets api
func GetSecretsAPISettings() APISetting {
	return apiEnvSetting[secretsKey]
}
