// +build !windows

package virtualenvironment

import (
	"os"
	"strings"
)

func inheritEnv(env map[string]string) map[string]string {
	for _, kv := range os.Environ() {
		eq := strings.Index(kv, "=")
		if eq < 0 {
			continue
		}
		key := kv[:eq]
		value := kv[eq+1:]
		if _, ok := env[key]; !ok {
			env[key] = value
		}
	}
	return env
}
