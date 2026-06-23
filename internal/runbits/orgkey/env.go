package orgkey

import "github.com/ActiveState/cli/internal/constants"

// stringConfigReader reads string-valued config options.
type stringConfigReader interface {
	GetString(key string) string
}

// SanitizeChildEnv removes private-ingredient key-service credentials from env
// so they are never propagated to child process environments.
func SanitizeChildEnv(cfg stringConfigReader, env map[string]string) {
	if tokenEnv := cfg.GetString(constants.PrivateIngredientBearerTokenEnvConfig); tokenEnv != "" {
		delete(env, tokenEnv)
	}
}
