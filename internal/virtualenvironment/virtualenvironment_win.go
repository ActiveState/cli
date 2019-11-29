// +build windows

package virtualenvironment

import (
	"os"
	"regexp"
	"strings"
)

func inheritEnv(env map[string]string) map[string]string {
	for _, kv := range os.Environ() {
		split := strings.Split(kv, "=")
		key := split[0]
		value := split[1]

		// Windows allows environment variables that are not uppercase.
		// This can lead to duplicate path entries. At this point we
		// have already constructed a PATH so it's safe to discard the
		// os Path.
		if strings.ToLower(key) == "path" {
			continue
		}

		// cmd.exe on Windows uses some dynamic environment variables
		// that begin with an '='. We want to make sure we include
		// these in the virtual environment. For more information see:
		// https://devblogs.microsoft.com/oldnewthing/20100506-00/?p=14133
		dynamicEnvVarRe := regexp.MustCompile(`(^=.+)=(.+)`)
		groups := dynamicEnvVarRe.FindStringSubmatch(kv)
		if len(groups) == 0 {
			continue
		}
		env[groups[1]] = groups[2]

		if _, ok := env[key]; !ok {
			env[key] = value
		}
	}

	return env
}
