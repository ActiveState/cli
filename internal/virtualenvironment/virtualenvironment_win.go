// +build windows

package virtualenvironment

import (
	"os"
	"regexp"
	"strings"
)

func inheritEnv(env map[string]string) map[string]string {
	dynamicEnvVarRe := regexp.MustCompile(`(^=.+)=(.+)`)

	for _, kv := range os.Environ() {
		split := strings.Split(kv, "=")
		key := split[0]
		value := split[1]

		// cmd.exe on Windows uses some dynamic environment variables
		// that begin with an '='. We want to make sure we include
		// these in the virtual environment. For more information see:
		// https://devblogs.microsoft.com/oldnewthing/20100506-00/?p=14133
		if strings.HasPrefix(kv, "=") {
			groups := dynamicEnvVarRe.FindStringSubmatch(kv)
			if len(groups) == 0 {
				continue
			}
			env[groups[1]] = groups[2]
		} else {
			// Windows allows environment variables that are not uppercase.
			// This can lead to duplicate path entries. At this point we
			// have already constructed the env vars that we need for
			// our virtual environment so we discard any duplicate entries`.
			if _, ok := env[strings.ToUpper(key)]; ok {
				continue
			}

			if _, ok := env[key]; !ok {
				env[key] = value
			}
		}
	}

	return env
}
