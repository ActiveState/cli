// +build windows

package activate

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ActiveState/cli/internal/virtualenvironment"
)

func envOutput() (string, error) {
	venv := virtualenvironment.Get()
	fail := venv.Activate()
	if fail != nil {
		return "", fail
	}

	env := virtualenvironment.Get().GetEnvSlice(true)
	envJSON := make([]string, len(env))
	dynamicEnvVarRe := regexp.MustCompile(`(^=.+)=(.+)`)
	var key, value string
	for i, kv := range env {
		if strings.HasPrefix(kv, "=") {
			groups := dynamicEnvVarRe.FindStringSubmatch(kv)
			if len(groups) < 3 {
				continue
			}
			key = groups[1]
			value = groups[2]
		} else {
			eq := strings.Index(kv, "=")
			if eq < 0 {
				continue
			}
			key = kv[:eq]
			value = kv[eq+1:]
		}
		envJSON[i] = fmt.Sprintf(
			"\"%s\": \"%s\"",
			strings.ReplaceAll(key, "\\", "\\\\"),
			strings.ReplaceAll(value, "\\", "\\\\"),
		)
	}

	return fmt.Sprintf("{ %s }", strings.Join(envJSON, ", ")), nil
}
