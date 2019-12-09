// +build !windows

package activate

import (
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/virtualenvironment"
)

func envOutput() (string, error) {
	venv := virtualenvironment.Get()
	fail := venv.Activate()
	if fail != nil {
		return "", fail
	}

	env := venv.GetEnvSlice(true)
	envJSON := make([]string, len(env))
	for i, kv := range env {
		eq := strings.Index(kv, "=")
		if eq < 0 {
			continue
		}
		envJSON[i] = fmt.Sprintf(
			"\"%s\": \"%s\"",
			strings.ReplaceAll(kv[:eq], "\\", "\\\\"),
			strings.ReplaceAll(kv[eq+1:], "\\", "\\\\"),
		)
	}

	return fmt.Sprintf("{ %s }", strings.Join(envJSON, ", ")), nil
}
