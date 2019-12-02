// +build !windows

package virtualenvironment

import (
	"os"
	"strings"
)

func inheritEnv(env map[string]string) map[string]string {
	for _, kv := range os.Environ() {
		split := strings.Split(kv, "=")
		key := split[0]
		value := split[1]
		if _, ok := env[key]; !ok {
			env[key] = value
		}
	}
	return env
}